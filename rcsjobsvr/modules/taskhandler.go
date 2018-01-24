package modules

import (
	"errors"
	"log"
	"os"
	"rcs/rcsagent/modules"
	"rcs/utils"
	//	"reflect"
	"net/url"
	"path/filepath"
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
	for v := range th.tasks {
		if task, ok := v.(*utils.RcsTaskReq); ok {
			log.Println("Got a task request:", task.Runid)
			once := new(sync.Once)
			for _, ip := range task.Targets {
				go th.handlerequest(task.Runid, ip, task.Atomicrequest, once) //对于一个任务中的多个agent进行并发处理；task.AtomicReq是一个interface(引用变量),非并发安全
			}
		}
	}

}
func (th *taskHandler) handlerequest(rid, ip string, req modules.Atomicrequest, once *sync.Once) {
	resp, err := th.rpccall(rid, ip, req, once)
	if err != nil {
		log.Println("th.rpccall:", err)
		return
	}
	th.resps <- resp
}
func (th *taskHandler) rpccall(rid string, ip string, req modules.Atomicrequest, once *sync.Once) (response *utils.RcsTaskResp, err error) {
	response = new(utils.RcsTaskResp)
	response.Runid = rid
	response.AgentIP = ip

	ai := th.getAgent(ip)
	if ai == nil { //广播模式下,后续优化为路由模式
		return nil, errors.New("agent is invalid in this jobsvr:" + ip)
	}
	//ai.doing.Lock() //针对同一ip的并发请求,加互斥锁，避免资源冲突；由于锁是与每个agent绑定的；不同ip的并发请求正常执行。
	//defer ai.doing.Unlock()
	rcli := ai.rpcli
	service := `Service.Call`
	var args modules.Atomicrequest

	resp := new(modules.Atomicresponse)
	var FileUrl, FileMd5 string
	dl_file := func() {
		if err := modules.Downloadfilefromurl(FileUrl, FileMd5, filepath.Join(th.fcdir, FileMd5)); err != nil {
			log.Println("file cached faild:", FileUrl, err)
			return
		}
		log.Println("file cached success:", FileUrl)
	}
	set_url4agent := func(FileUrl, FileMd5 string) (string, error) {
		u, e := url.Parse(FileUrl)
		if e != nil {
			return "", e
		}
		filename := u.Query().Get("rename")
		if filename == "" {
			filename = filepath.Base(strings.Split(u.RequestURI(), "?")[0])
		}
		if filename == "" {
			return "", errors.New("srcfileurl is invalid:" + FileUrl)
		}
		//给每个agent的url地址可能不一样，因为agent可能从内网连过来，也可能从外网连过来
		jobsvrip := strings.Split(ai.conn.LocalAddr().String(), ":")[0]
		u.Host = jobsvrip + ":" + strings.Split(th.fcaddr, ":")[1]
		u.Path = "/" + th.fcdir + "/" + FileMd5 + "/" + filename
		return u.String(), nil
	}
	switch v := req.(type) { //just two Atomicrequest type should be download the file
	case *modules.Cmd_script_req:
		FileUrl = v.FileUrl
		FileMd5 = v.FileMd5
		once.Do(dl_file)
		newurl, err := set_url4agent(FileUrl, FileMd5)
		if err != nil {
			response.Flag = false
			response.Result = err.Error()
			return response, nil
		}
		v.FileUrl = newurl
		args = v
	case *modules.File_push_req:
		FileUrl = v.Sfileurl
		FileMd5 = v.Sfilemd5
		once.Do(dl_file)
		newurl, err := set_url4agent(FileUrl, FileMd5)
		if err != nil {
			response.Flag = false
			response.Result = err.Error()
			return response, nil
		}
		v.Sfileurl = newurl
		args = v
	default:
		args = req
	}
	divcall := rcli.Go(service, &args, resp, nil) //异步调用并设置超时时间
	select {
	case replaycall := <-divcall.Done:
		if replaycall.Error != nil {
			resp.Result = resp.Result + replaycall.Error.Error()
		}
	case <-time.After(time.Second * time.Duration(th.rpcto)):
		resp.Result = "Atomicrequest call:timeout:"
	}
	response.Flag = resp.Flag
	response.Result = resp.Result
	return response, nil
}
