package main

import (
	"encoding/gob"
	"io"
	"log"
	"os"
	agentmod "rcs/rcsagent/modules"
	"rcs/rcsjobsvr/rcsjobtransfer/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"
)

var (
	masterCKT  int
	masterAddr string
)
var logfile *os.File
var prod_taskreq, coms_taskresp, coms_agentinfo *utils.Pdcser

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
	logfile, err = os.OpenFile("log/jobtransfer.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}
	//log.SetOutput(logfile)
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	//log.SetOutput(logfile)
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	//处理配置文件
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
masterCKT          = 3
masterAddr         = 127.0.0.1:9525
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

	cf := utils.HandleConfigFile("cfg/jobtransfer.ini", defcfg)

	masterCKT = cf.MustInt("BASE", "masterCKT")
	masterAddr = cf.MustValue("BASE", "masterAddr")
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

	prod_taskreq, err = utils.Newpdcser(taskreq_mqUri, taskreq_exChangeName, taskreq_queueName, taskreq_rKey)
	if err != nil {
		log.Fatalln(err)
	}
	coms_taskresp, err = utils.Newpdcser(taskresp_mqUri, taskresp_exChangeName, taskresp_queueName, taskresp_rKey)
	if err != nil {
		log.Fatalln(err)
	}
	coms_agentinfo, err = utils.Newpdcser(agentsync_mqUri, agentsync_exChangeName, agentsync_queueName, agentsync_rKey)
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
	defer prod_taskreq.Close()
	defer coms_taskresp.Close()
	defer coms_agentinfo.Close()

	tfer := modules.Newtransfer(prod_taskreq, coms_taskresp, coms_agentinfo)
	if tfer == nil {
		log.Fatalln("invalid transfer")
	}
	if _, mms := utils.NewTClient(masterAddr, masterCKT, 0, true, tfer); mms != nil {
		log.Fatalln(mms.Connect())
	}

}
