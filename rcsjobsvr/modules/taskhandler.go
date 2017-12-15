package modules

import (
	"errors"
	"log"
	"os"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"runtime/debug"
	"sync"
	"time"
)

type taskHandler struct {
	rpcto         int
	fcdir, fcaddr string
	tasks         <-chan interface{}
	resps         chan<- *utils.RcsTaskResp
	getAgent      func(string) *agentEntry
	setUrlPending *sync.Mutex
}

func NewTaskHandler(rpctimeout int, filecdir, filecaddr string, tchan <-chan interface{}, respchan chan<- *utils.RcsTaskResp, getfunc func(string) *agentEntry) *taskHandler {
	return &taskHandler{
		rpcto:         rpctimeout,
		fcdir:         filecdir,
		fcaddr:        filecaddr,
		tasks:         tchan,
		resps:         respchan,
		getAgent:      getfunc,
		setUrlPending: new(sync.Mutex),
	}
}
func (th *taskHandler) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()

	for v := range th.tasks {
		if task, ok := v.(*utils.RcsTaskReq); ok {
			log.Println("Got a task request:", task.Runid)
			for _, ip := range task.Targets {
				go th.handlerequest(task.Runid, ip, task.Atomicrequest) //对于一个任务中的多个agent进行并发处理；task.AtomicReq是一个interface(引用变量),非并发安全
			}
		}
	}

}
func (th *taskHandler) handlerequest(rid, ip string, req modules.Atomicrequest) {
	resp, err := th.rpccall(rid, ip, req)
	if err != nil {
		log.Println("Rpc call:", err)
		return
	}
	th.resps <- resp
}
func (th *taskHandler) rpccall(rid string, ip string, req modules.Atomicrequest) (response *utils.RcsTaskResp, err error) {
	response = new(utils.RcsTaskResp)
	response.Runid = rid
	response.AgentIP = ip

	ai := th.getAgent(ip)
	if ai == nil { //广播模式下,后续优化为路由模式
		return nil, errors.New("agent is invalid in this jobsvr:" + ip)
	}
	ai.doing.Lock() //针对同一ip的并发请求,加互斥锁，避免资源冲突；由于锁是与每个agent绑定的；不同ip的并发请求正常执行。
	defer ai.doing.Unlock()
	rcli := ai.rpcli
	service := `Service.Call`
	args := req
	resp := new(modules.Atomicresponse)

	divcall := rcli.Go(service, &args, resp, nil) //异步调用并设置超时时间
	th.setUrlPending.Unlock()
	select {
	case replaycall := <-divcall.Done:
		if replaycall.Error != nil {
			resp.Result = resp.Result + replaycall.Error.Error()
			log.Println("Rpc call:", replaycall.Error.Error())
		}
	case <-time.After(time.Second * time.Duration(th.rpcto)):
		resp.Result = "rpc call timeout"
	}
	response.Flag = resp.Flag
	response.Result = resp.Result
	return response, nil
}
