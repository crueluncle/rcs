package modules

import (
	"errors"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"rcs/rcsagent"
	"rcs/utils"
	"runtime/debug"
	"strings"
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

	var FileUrl, FileMd5 string
	var dlf_err error

	for v := range th.tasks {
		if task, ok := v.(*utils.RcsTaskReq); ok {
			log.Println("Got a task request:", task.Runid)

			FileUrl = task.AtomicReq.GetFileUrl()
			FileMd5 = task.AtomicReq.GetFileMd5()
			if FileUrl != "" && FileMd5 != "" {
				dlf_err = rcsagent.Downloadfilefromurl(FileUrl, FileMd5, filepath.Join(th.fcdir, FileMd5))
			}
			for _, ip := range task.Targets {
				go th.handlerequest(task.Runid, ip, task.AtomicReq, dlf_err) //对于一个任务中的多个agent进行并发处理；task.AtomicReq是一个interface(引用变量),非并发安全
			}
		}
	}

}
func (th *taskHandler) handlerequest(rid, ip string, req rcsagent.RpcCallRequest, dlf_err error) {
	resp, err := th.rpccall(rid, ip, req, dlf_err)
	if err != nil {
		log.Println("Rpc call:", err)
		return
	}
	th.resps <- resp
}
func (th *taskHandler) rpccall(rid string, ip string, req rcsagent.RpcCallRequest, dlf_err error) (response *utils.RcsTaskResp, err error) {

	response = new(utils.RcsTaskResp)
	response.Runid = rid
	response.AgentIP = ip

	ai := th.getAgent(ip)
	if ai == nil {
		return nil, errors.New("agent is invalid in this jobsvr:" + ip)
	}
	ai.doing.Lock()
	defer ai.doing.Unlock()
	rcli := ai.rpcli
	service := `ModuleService.Run`
	args := req
	resp := new(rcsagent.RpcCallResponse)

	if dlf_err != nil {
		response.Flag = false
		response.Result = dlf_err.Error()
		return response, nil
	}

	FileUrl := args.GetFileUrl()
	FileMd5 := args.GetFileMd5()

	if FileUrl != "" && FileMd5 != "" {
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

		jobsvrip := strings.Split(ai.conn.LocalAddr().String(), ":")[0]

		u.Host = jobsvrip + ":" + strings.Split(th.fcaddr, ":")[1]
		u.Path = "/" + th.fcdir + "/" + FileMd5 + "/" + filename
		th.setUrlPending.Lock()
		args.SetFileUrl(u.String())
	}
	divcall := rcli.Go(service, &args, resp, nil)
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
