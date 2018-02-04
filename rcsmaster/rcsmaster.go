package main

/*
1.TH_JobsvrManage  管理连接上来的jobsvr保存，并周期性探测状态；
2.TH_RecvRespFromJobsvr 从jobsvr中循环接收响应消息，保存到redis:redis作为master内部临时数据存储方式,无需持久化,1天自动过期;数据的持久化由前端调用者自行处理
3.TH_RcshttpAPI 提供外部接口单独协程,整个rcs系统对外只有1个api,异步调用
	 POST  http://127.0.0.1:9999/runtask
*/
import (
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"os"
	agentmod "rcs/rcsagent/modules"
	"rcs/rcsmaster/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"
	//	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	masterAddr,
	apiServer_addr,
	redisconstr, syncredisconstr,
	redispass, syncredispass string
	redisDB, syncredisDB, //redis DB
	rMaxIdle, syncrMaxIdle, //redis连接池最大空闲连接
	rMaxActive, syncrMaxActive int //redis连接池最大连接数
	mqUri,
	exChangeName,
	queueName,
	rKey string
)
var logfile *os.File
var redisClient1, redisClient2 *redis.Pool
var comsumer *utils.Pdcser
var msgs = make(chan []byte, 64)
var taskList chan *utils.RcsTaskReq

func init() {
	gob.Register(utils.KeepaliveMsg{})
	gob.Register(utils.RcsTaskResp{})
	gob.Register(utils.RcsTaskReq{})
	gob.Register(utils.AgentSyncMsg{})
	gob.Register(&agentmod.File_push_req{})
	gob.Register(&agentmod.File_pull_req{})
	gob.Register(&agentmod.File_cp_req{})
	gob.Register(&agentmod.File_del_req{})
	gob.Register(&agentmod.File_rename_req{})
	gob.Register(&agentmod.File_grep_req{})
	gob.Register(&agentmod.File_replace_req{})
	gob.Register(&agentmod.File_mreplace_req{})
	gob.Register(&agentmod.File_md5sum_req{})
	gob.Register(&agentmod.File_ckmd5sum_req{})
	gob.Register(&agentmod.File_zip_req{})
	gob.Register(&agentmod.File_unzip_req{})
	gob.Register(&agentmod.Cmd_script_req{})
	gob.Register(&agentmod.Cmd_run_req{})
	gob.Register(&agentmod.Os_restart_req{})
	gob.Register(&agentmod.Os_shutdown_req{})
	gob.Register(&agentmod.Os_setpwd_req{})
	gob.Register(&agentmod.Firewall_set_req{})
	gob.Register(&agentmod.Process_stop_req{})
	gob.Register(&agentmod.Rcs_ping_req{})
	utils.MsgTypeRegist(&utils.RcsTaskResp{})
	utils.MsgTypeRegist(&utils.KeepaliveMsg{})
	utils.MsgTypeRegist(&utils.AgentSyncMsg{})
	var errs error
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	logfile, errs = os.OpenFile("log/rcsmaster.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	//log.SetOutput(logfile)
	//	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	//log.SetOutput(logfile)

	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
masterAddr         = 0.0.0.0:9525
[RespRedis]
redisconstr = 127.0.0.1:6379
redisDB = 0
redispass   = yourPassword
rMaxIdle    = 100
rMaxActive  = 20000
[SyncRedis]
redisconstr = 127.0.0.1:6379
redisDB = 1
redispass   = yourPassword
rMaxIdle    = 100
rMaxActive  = 1000
[TaskMq]
mqUri = amqp://admin:admin@127.0.0.1:5672/
exChangeName = rcs
queueName = task
rKey = task.msg`
	cf := utils.HandleConfigFile("cfg/rcsmaster.ini", defcfg)
	masterAddr = cf.MustValue("BASE", "masterAddr")

	redisconstr = cf.MustValue("RespRedis", "redisconstr")
	redisDB = cf.MustInt("RespRedis", "redisDB")
	redispass = cf.MustValue("RespRedis", "redispass")
	rMaxIdle = cf.MustInt("RespRedis", "rMaxIdle")
	rMaxActive = cf.MustInt("RespRedis", "rMaxActive")

	syncredisconstr = cf.MustValue("SyncRedis", "redisconstr")
	syncredisDB = cf.MustInt("SyncRedis", "redisDB")
	syncredispass = cf.MustValue("SyncRedis", "redispass")
	syncrMaxIdle = cf.MustInt("SyncRedis", "rMaxIdle")
	syncrMaxActive = cf.MustInt("SyncRedis", "rMaxActive")

	mqUri = cf.MustValue("TaskMq", "mqUri")
	exChangeName = cf.MustValue("TaskMq", "exChangeName")
	queueName = cf.MustValue("TaskMq", "queueName")
	rKey = cf.MustValue("TaskMq", "rKey")

	taskList = make(chan *utils.RcsTaskReq, 64)
	redisClient1, errs = utils.Newredisclient(redisconstr, redispass, redisDB, rMaxIdle, rMaxActive) //for write response msg
	if errs != nil {
		log.Fatalln(errs)
	}
	redisClient2, errs = utils.Newredisclient(syncredisconstr, syncredispass, syncredisDB, syncrMaxIdle, syncrMaxActive) //for write agentsync msg
	if errs != nil {
		log.Fatalln(errs)
	}
	comsumer, errs = utils.Newpdcser(mqUri, exChangeName, queueName, rKey)
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
	defer comsumer.Close()
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	go func() {
		var (
			taskjson = new(utils.RcsTaskReqJson)
			task     *utils.RcsTaskReq
			err      error
		)
		for taskdata := range msgs {
			err = json.Unmarshal(taskdata, taskjson)
			if err != nil {
				log.Println(err)
				continue
			}
			task, err = taskjson.Parse()
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("Fetch a task from mq:", task.Runid)
			taskList <- task
		}
	}()
	go func() {
		log.Fatalln(comsumer.Comsumer(msgs))
	}()

	var jobManagerSvr = modules.NewJobsvrManager(redisClient1.Get, redisClient2.Get, taskList)
	if _, ts := utils.NewTServer(masterAddr, jobManagerSvr); ts != nil {
		log.Fatalln(ts.Serve())
	}

}
