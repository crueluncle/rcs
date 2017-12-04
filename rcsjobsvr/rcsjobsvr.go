package main

/*4个goroutine
TH_AgentManage() 用于管理agent的注册、注销和周期性心跳检测
TH_masterManage() 与master建立一条tcp连接用于与master的交互,并周期性心跳检测
TH_HandleTasks 循环读取Tasks任务队列,处理每一个任务
TH_FileSvr 提供文件下载服务,http服务
*/
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

const filecachedir string = `JobsrvFileCacheDir` //文件缓存服务器根目录
var (
	routeId     uint16 //for identify  this jobsvr,must be uniq between all jobsvrs ,less than 65535
	rpcTimeOut, //rpc调用超时时间:包括rpcclient与rpcserver之间网络通信的时间、rpc服务函数本身的执行时间;FYI:对一个rpcclient的并发rpc调用实际是串行执行的,因多个goroutine对同一个net.conn的数据传输是串行化的，因此每次rpc调用的时间消耗实际是不断增长的
	agentCKT, //agent状态检测时间间隔(1-10s之间的随机时间),agent量较多，采用随机检测避免风暴
	masterCKT, //master状态检测时间间隔(s),jobsvr数量较少,10s高频检测,保证一定的实时可用性
	taskLength int //任务缓冲队列长度
	jobsvrAddr, //与agent交互jobsvr需要监听的地址,跑rpc
	masterAddr, //master地址
	masterSyncAddr, //master路由同步地址
	filecacheAddr string //rekjobsvr传输文件给agent,ip只能是0.0.0.0,跑http,用于替换fileregistryip
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
