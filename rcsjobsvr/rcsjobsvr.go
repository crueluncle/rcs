package main

import (
	"encoding/gob"
	"io"
	"log"
	"os"
	"rcs/rcsagent"
	"rcs/rcsjobsvr/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"
	"sync"
)

const filecachedir string = `JobsrvFileCacheDir`

var (
	routeId uint16 //for identify  this jobsvr,must be uniq between all jobsvrs ,less than 65535
	rpcTimeOut,
	agentCKT,
	masterCKT,
	taskLength int
	jobsvrAddr,
	masterAddr,
	masterSyncAddr,
	filecacheAddr string
)
var logfile *os.File

var tasks chan interface{}
var resps chan *utils.RcsTaskResp
var nodeRouteMap *sync.Map

func init() { //初始化操作
	utils.MsgTypeRegist(&utils.RcsTaskReq{})
	gob.Register(&rcsagent.Script_Run_Req{})
	gob.Register(&rcsagent.File_Push_Req{})
	gob.Register(&rcsagent.Rcs_Restart_Req{})
	gob.Register(&rcsagent.Rcs_Stop_Req{})
	gob.Register(&rcsagent.Rcs_Upgrade_Req{})
	gob.Register(&rcsagent.Rcs_HeartBeat_Req{})
	gob.Register(rcsagent.RpcCallResponse{})
	gob.Register(utils.KeepaliveMsg{})
	gob.Register(utils.RcsTaskResp{})
	//处理日志
	var err error
	logfile, err = os.OpenFile("log/rcsjobsvr.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	//log.SetOutput(logfile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	//log.SetOutput(logfile)
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	//处理配置文件
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
routeId			   = 10000
rpcTimeOut         = 3600                  
agentCKT           = 10                    
masterCKT          = 10                    
taskLength         = 1280                    
jobsvrAddr         = 0.0.0.0:9529      
masterAddr         = 127.0.0.1:9525
masterSyncAddr     = 127.0.0.1:9526        
filecacheAddr      = 0.0.0.0:9530`

	cf := utils.HandleConfigFile("cfg/rcsjobsvr.ini", defcfg)
	routeId = uint16(cf.MustInt("BASE", "routeId"))
	rpcTimeOut = cf.MustInt("BASE", "rpcTimeOut")
	agentCKT = cf.MustInt("BASE", "agentCKT")
	masterCKT = cf.MustInt("BASE", "masterCKT")
	taskLength = cf.MustInt("BASE", "taskLength")
	jobsvrAddr = cf.MustValue("BASE", "jobsvrAddr")
	masterAddr = cf.MustValue("BASE", "masterAddr")
	filecacheAddr = cf.MustValue("BASE", "filecacheAddr")
	masterSyncAddr = cf.MustValue("BASE", "masterSyncAddr")

	tasks = make(chan interface{}, taskLength)
	resps = make(chan *utils.RcsTaskResp, taskLength)
	nodeRouteMap = new(sync.Map)
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()
	runtime.GOMAXPROCS(runtime.NumCPU() * 4)
	defer logfile.Close()
	defer close(tasks)
	defer close(resps)

	myagentManager := modules.NewAgentMngSvr(agentCKT, routeId, nodeRouteMap)
	myfileServer := modules.NewFileSvr(filecacheAddr, filecachedir)
	mymasterMngSvr := modules.NewMasterManager(tasks, resps)
	mytaskhandler := modules.NewTaskHandler(rpcTimeOut, filecachedir, filecacheAddr, tasks, resps, myagentManager.Getagent)
	//	mySyncRmap := modules.NewRouteSynchronizer(nodeRouteMap)

	go myfileServer.ServeFile()

	go func() {
		if _, ams := utils.NewTServer(jobsvrAddr, myagentManager); ams != nil {
			log.Fatalln(ams.Serve())
		}
	}()

	go func() {
		if _, mms := utils.NewTClient(masterAddr, masterCKT, 0, true, mymasterMngSvr); mms != nil {
			log.Fatalln(mms.Connect())
		}
	}()
	/*go func() {
		if _, mms := utils.NewTClient(masterSyncAddr, masterCKT, 0, true, mySyncRmap); mms != nil {
			log.Fatalln(mms.Connect())
		}
	}()*/
	mytaskhandler.Run()
}
