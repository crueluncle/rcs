package main

import (
	"encoding/gob"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"runtime/debug"
	"syscall"
	//"github.com/Unknwon/goconfig"
)

var logf *os.File
var (
	RconT         int    //agent断开jobsvr连接后，在多长的随机时间内重连jobsvr,agent数量可能较多，随机重连避免风暴
	JobsvrAddr    string // jobsvr地址
	FilecacheAddr string //filecacheSvr地址
)

func init() {
	gob.Register(&modules.File_push_req{})
	gob.Register(&modules.File_pull_req{})
	gob.Register(&modules.File_cp_req{})
	gob.Register(&modules.File_del_req{})
	gob.Register(&modules.File_grep_req{})
	gob.Register(&modules.File_replace_req{})
	gob.Register(&modules.File_mreplace_req{})
	gob.Register(&modules.File_md5sum_req{})
	gob.Register(&modules.File_ckmd5sum_req{})
	gob.Register(&modules.File_zip_req{})
	gob.Register(&modules.File_unzip_req{})
	gob.Register(&modules.Cmd_script_req{})
	gob.Register(&modules.Cmd_run_req{})
	gob.Register(&modules.Os_restart_req{})
	gob.Register(&modules.Os_shutdown_req{})
	gob.Register(&modules.Os_setpwd_req{})
	gob.Register(&modules.Firewall_set_req{})
	gob.Register(&modules.Process_stop_req{})
	gob.Register(&modules.Rcs_ping_req{})
	file, _ := exec.LookPath(os.Args[0])
	path := filepath.Dir(file)
	if err := os.MkdirAll(filepath.Join(path, `log`), 0755); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(filepath.Join(path, `cfg`), 0755); err != nil {
		log.Fatalln(err)
	}
	logfilename := filepath.Join(path, `log/rcsagent.log`)
	logf, _ = os.OpenFile(logfilename, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0777)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(logf)
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)

	inifilename := filepath.Join(path, `cfg/rcsagent.ini`)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
	[BASE]
	rconT             = 10
	jobsvrAddr        = 127.0.0.1:9529
	filecacheAddr     = 127.0.0.1:9530`
	cf := utils.HandleConfigFile(inifilename, defcfg)
	RconT = cf.MustInt("BASE", "rconT")
	JobsvrAddr = cf.MustValue("BASE", "jobsvrAddr")
	FilecacheAddr = cf.MustValue("BASE", "filecacheAddr")
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()
	defer logf.Close()

	var e error
	var tc *utils.TClient
	var agentServe utils.TFunc = modules.InitRPCserver

	if e, tc = utils.NewTClient(JobsvrAddr, RconT, 0, true, agentServe); tc != nil {
		log.Fatalln(tc.Connect())
	}
	log.Fatalln(e)
}
