package main

/*
1.TH_JobsvrManage  管理jobsvr:将连接上来的jobsvr保存起来，并周期性探测状态；
2.TH_RecvRespFromJobsvr 并从jobsvr中循环接收响应消息，保存到redis:redis作为master内部临时数据存储方式,无需持久化,1天自动过期;数据的持久化由前端调用者自行处理
3.TH_RcshttpAPI 提供外部接口单独协程,整个rek系统对外只有1个api
	 POST  http://127.0.0.1:9999/runtask                       //提交任务json串到master,解析为为Rcstask对象并发送给jobsvr,给调用方返回json结构
4.整套系统是通过ip地址来唯一标识OS,因此对于有多个孤岛内网的环境,须满足内网ip唯一性(统一规划)
*/
import (
	"encoding/gob"
	"io"
	"log"
	"os"
	"rcs/rcsmaster/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"

	"github.com/garyburd/redigo/redis"
)

var (
	masterAddr, //master与jobsvr控制连接需要监听的地址
	apiServer_addr, //master对外提供task提交服务需要监听的地址
	redisconstr, //redis连接地址
	redispass string //redis认证密码
	redisDB, //redis DB
	rMaxIdle, //redis连接池最大空闲连接
	rMaxActive int //redis连接池最大连接数

)
var logfile *os.File
var redisClient *redis.Pool
var taskList chan *utils.RcsTaskReq

func init() {
	gob.Register(utils.KeepaliveMsg{})
	gob.Register(utils.RcsTaskResp{})
	gob.Register(utils.RcsTaskReq{})
	utils.MsgTypeRegist(&utils.RcsTaskResp{})
	utils.MsgTypeRegist(&utils.KeepaliveMsg{})
	var errs error
	logfile, errs = os.OpenFile("log/rcsmaster.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	//log.SetOutput(logfile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	//log.SetOutput(logfile)

	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
masterAddr         = 0.0.0.0:9525
apiServer_addr     = 0.0.0.0:9527
redisconstr = 127.0.0.1:6379
redisDB     = 0
redispass   = yourPassword
rMaxIdle    = 100
rMaxActive  = 20000`
	cf := utils.HandleConfigFile("cfg/rcsmaster.ini", defcfg)
	masterAddr = cf.MustValue("BASE", "masterAddr")
	apiServer_addr = cf.MustValue("BASE", "apiServer_addr")
	redisconstr = cf.MustValue("BASE", "redisconstr")
	redisDB = cf.MustInt("BASE", "redisDB")
	redispass = cf.MustValue("BASE", "redispass")
	rMaxIdle = cf.MustInt("BASE", "rMaxIdle")
	rMaxActive = cf.MustInt("BASE", "rMaxActive")

	taskList = make(chan *utils.RcsTaskReq, 64)
	redisClient, errs = utils.Newredisclient(redisconstr, redispass, redisDB, rMaxIdle, rMaxActive)
	if errs != nil {
		log.Fatalln(errs)
	}
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()
	defer logfile.Close()
	runtime.GOMAXPROCS(runtime.NumCPU() * 3)

	var jobManagerSvr = modules.NewJobsvrManager(redisClient.Get, taskList)
	var apiserver = modules.NewMasterapi(apiServer_addr, taskList)

	go func() {
		apiserver.Serve()
	}()

	if _, ts := utils.NewTServer(masterAddr, jobManagerSvr); ts != nil {
		log.Fatalln(ts.Serve())
	}

}
