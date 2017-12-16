package modules

import (
	"errors"
	"log"
	"os"
	"rcs/rcsagent/modules"
	"rcs/utils"
	//	"reflect"
	"path/filepath"
	"runtime/debug"

	"time"
)

type taskHandler struct {
	rpcto         int
	fcdir, fcaddr string
	tasks         <-chan interface{}
	resps         chan<- *utils.RcsTaskResp
	getAgent      func(string) *agentEntry
}

func NewTaskHandler(rpctimeout int, filecdir, filecaddr string, tchan <-chan interface{}, respchan chan<- *utils.RcsTaskResp, getfunc func(string) *agentEntry) *taskHandler {
	return &taskHandler{
		rpcto:    rpctimeout,
		fcdir:    filecdir,
		fcaddr:   filecaddr,
		tasks:    tchan,
		resps:    respchan,
		getAgent: getfunc,
	}
}
func (th *taskHandler) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	var FileUrl, FileMd5 string
	var dlf_err error
	for v := range th.tasks {
		if task, ok := v.(*utils.RcsTaskReq); ok {
			log.Println("Got a task request:", task.Runid)
			switch v := task.Atomicrequest.(type) { //just two Atomicrequest type should be download the file
			case modules.Cmd_script_req:
				FileUrl = v.FileUrl
				FileMd5 = v.FileMd5
			case modules.File_push_req:
				FileUrl = v.Sfileurl
				FileMd5 = v.Sfilemd5
			default:
			}
			if FileUrl != "" && FileMd5 != "" {
				dlf_err = modules.Downloadfilefromurl(FileUrl, FileMd5, filepath.Join(th.fcdir, FileMd5))
			}
			for _, ip := range task.Targets {
				go th.handlerequest(task.Runid, ip, task.Atomicrequest, dlf_err) //对于一个任务中的多个agent进行并发处理；task.AtomicReq是一个interface(引用变量),非并发安全
			}
		}
	}

}
func (th *taskHandler) handlerequest(rid, ip string, req modules.Atomicrequest, dlf_err error) {
	resp, err := th.rpccall(rid, ip, req, dlf_err)
	if err != nil {
		log.Println("Rpc call:", err)
		return
	}
	th.resps <- resp
}
func (th *taskHandler) rpccall(rid string, ip string, req modules.Atomicrequest, dlf_err error) (response *utils.RcsTaskResp, err error) {
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
	if dlf_err != nil { //文件下载失败,直接返回
		response.Flag = false
		response.Result = dlf_err.Error()
		return response, nil
	}
	/*
		FileUrl := args.GetFileUrl()
		FileMd5 := args.GetFileMd5()

		if FileUrl != "" && FileMd5 != "" { //实际是Script_Run_Req或File_Push_Req两种请求
			u, e := url.Parse(FileUrl)
			if e != nil {
				log.Println(e)
				response.Flag = false
				response.Result = e.Error()
				return response, nil
			}
			filename := u.Query().Get("rename")
			if filename == "" {
				filename = filepath.Base(strings.Split(u.RequestURI(), "?")[0])
			}
			if filename == "" {
				response.Flag = false
				response.Result = "srcfileurl is invalid:" + FileUrl
				return response, nil
			}
			//给每个agent的url地址可能不一样，因为agent可能从内网连过来，也可能从外网连过来
			jobsvrip := strings.Split(ai.conn.LocalAddr().String(), ":")[0]
			//log.Println("jobsvrip:", rid, jobsvrip)
			u.Host = jobsvrip + ":" + strings.Split(th.fcaddr, ":")[1]
			u.Path = "/" + th.fcdir + "/" + FileMd5 + "/" + filename
			th.setUrlPending.Lock() //由于RpcCallRequest是一个interface(引用变量)，非并发安全,因此在改变变量时要加锁,实际这里降低了一定的并发性能
			args.SetFileUrl(u.String())
		}
	*/
	switch v := args.(type) { //just two Atomicrequest type should be reset the fileurl
	case modules.Cmd_script_req:

	case modules.File_push_req:

	default:
	}
	divcall := rcli.Go(service, &args, resp, nil) //异步调用并设置超时时间
	select {
	case replaycall := <-divcall.Done:
		if replaycall.Error != nil {
			resp.Result = resp.Result + replaycall.Error.Error()
			log.Println("Atomicrequest call:", replaycall.Error.Error())
		}
	case <-time.After(time.Second * time.Duration(th.rpcto)):
		resp.Result = "Atomicrequest call:timeout:"
	}
	response.Flag = resp.Flag
	response.Result = resp.Result
	return response, nil
}
