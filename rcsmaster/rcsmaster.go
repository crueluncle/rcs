package main

import (
	"encoding/gob"
	"io"
	"log"
	"os"
	"rcs/rcsagent"
	"rcs/rcsmaster/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"

	"github.com/garyburd/redigo/redis"
)

var (
	masterAddr,
	apiServer_addr,
	redisconstr,
	redispass string
	redisDB,
	rMaxIdle,
	rMaxActive int
)
var logfile *os.File
var redisClient *redis.Pool
var taskList chan *utils.RcsTaskReq

func init() {
	gob.Register(&rcsagent.Script_Run_Req{})
	gob.Register(&rcsagent.File_Push_Req{})
	gob.Register(&rcsagent.Rcs_Restart_Req{})
	gob.Register(&rcsagent.Rcs_Stop_Req{})
	gob.Register(&rcsagent.Rcs_Upgrade_Req{})
	gob.Register(&rcsagent.Rcs_HeartBeat_Req{})
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
