package main

import (
	"encoding/gob"
	"io"
	"log"
	"os"
	agentmod "rcs/rcsagent/modules"
	"rcs/rcsjobsvr/rcsagentmng/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"
)

var (
	routeId uint16 //for identify  this jobsvr,must be uniq between all jobsvrs ,less than 65535
	rpcTimeOut,
	agentCKT int
	jobsvrAddr string
	filecacheAddr,
	filecachedir string //文件缓存服务器根目录,配置rcsjobfilecache进程中的filecachedir
	logfile *os.File
)
var coms_taskreq, prods_taskresp, prods_agentinfo *utils.Pdcser

func init() { //初始化操作
	utils.MsgTypeRegist(&utils.RcsTaskReq{})
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
	//处理日志
	var err error
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	logfile, err = os.OpenFile("log/agentmng.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	//log.SetOutput(logfile)
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	//处理配置文件
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
routeId			   = 10000
rpcTimeOut         = 3600
agentCKT           = 20
jobsvrAddr         = 0.0.0.0:9529
filecacheAddr      = 0.0.0.0:9530
filecachedir = JobsrvFileCacheDir
[taskreq_mq]
mqUri = amqp://admin:admin@127.0.0.1:5672/
exChangeName = job
queueName = taskreq
rKey = task.req
[taskresp_mq]
mqUri = amqp://admin:admin@127.0.0.1:5672/
exChangeName = job
queueName = taskresp
rKey  = task.resp
[agentsync_mq]
mqUri = amqp://admin:admin@127.0.0.1:5672/
exChangeName = job
queueName = agentsync
rKey  = agent.sync`

	cf := utils.HandleConfigFile("cfg/agentmng.ini", defcfg)
	routeId = uint16(cf.MustInt("BASE", "routeId"))
	rpcTimeOut = cf.MustInt("BASE", "rpcTimeOut")
	agentCKT = cf.MustInt("BASE", "agentCKT")
	jobsvrAddr = cf.MustValue("BASE", "jobsvrAddr")
	filecacheAddr = cf.MustValue("BASE", "filecacheAddr")
	filecachedir = cf.MustValue("BASE", "filecachedir")

	taskreq_mqUri := cf.MustValue("taskreq_mq", "mqUri")
	taskreq_exChangeName := cf.MustValue("taskreq_mq", "exChangeName")
	taskreq_queueName := cf.MustValue("taskreq_mq", "queueName")
	taskreq_rKey := cf.MustValue("taskreq_mq", "rKey")
	taskresp_mqUri := cf.MustValue("taskresp_mq", "mqUri")
	taskresp_exChangeName := cf.MustValue("taskresp_mq", "exChangeName")
	taskresp_queueName := cf.MustValue("taskresp_mq", "queueName")
	taskresp_rKey := cf.MustValue("taskresp_mq", "rKey")
	agentsync_mqUri := cf.MustValue("agentsync_mq", "mqUri")
	agentsync_exChangeName := cf.MustValue("agentsync_mq", "exChangeName")
	agentsync_queueName := cf.MustValue("agentsync_mq", "queueName")
	agentsync_rKey := cf.MustValue("agentsync_mq", "rKey")

	coms_taskreq, err = utils.Newpdcser(taskreq_mqUri, taskreq_exChangeName, taskreq_queueName, taskreq_rKey)
	if err != nil {
		log.Fatalln(err)
	}
	prods_taskresp, err = utils.Newpdcser(taskresp_mqUri, taskresp_exChangeName, taskresp_queueName, taskresp_rKey)
	if err != nil {
		log.Fatalln(err)
	}
	prods_agentinfo, err = utils.Newpdcser(agentsync_mqUri, agentsync_exChangeName, agentsync_queueName, agentsync_rKey)
	if err != nil {
		log.Fatalln(err)
	}
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()
	runtime.GOMAXPROCS(runtime.NumCPU() * 4)
	defer logfile.Close()
	defer coms_taskreq.Close()
	defer prods_taskresp.Close()
	defer prods_agentinfo.Close()

	myagentManager := modules.NewAgentMngSvr(agentCKT, prods_agentinfo)

	mytaskhandler := modules.NewTaskHandler(rpcTimeOut, filecachedir, filecacheAddr, coms_taskreq, prods_taskresp, myagentManager.Getagent)
	//	mySyncRmap := modules.NewRouteSynchronizer(nodeRouteMap)

	go func() {
		if _, ams := utils.NewTServer(jobsvrAddr, myagentManager); ams != nil {
			log.Fatalln(ams.Serve())
		}
	}()

	mytaskhandler.Run()
}
