package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	//	"strings"
	"rcs/rcsagent"
	"rcs/utils"
	"syscall"

	"encoding/gob"

	//"github.com/Unknwon/goconfig"
)

var (
	rconT      int
	jobsvrAddr string
)
var logf *os.File

func init() {
	gob.Register(&rcsagent.Script_Run_Req{})
	gob.Register(&rcsagent.File_Push_Req{})
	gob.Register(&rcsagent.Rcs_Restart_Req{})
	gob.Register(&rcsagent.Rcs_Stop_Req{})
	gob.Register(&rcsagent.Rcs_Upgrade_Req{})
	gob.Register(&rcsagent.Rcs_HeartBeat_Req{})
	file, _ := exec.LookPath(os.Args[0])

	path := filepath.Dir(file)
	logfilename := filepath.Join(path, `log/rcsagent.log`)
	logf, _ = os.OpenFile(logfilename, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0777)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(logf)
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)

	inifilename := filepath.Join(path, `cfg/rcsagent.ini`)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
	[BASE]
	rconT             = 10
	jobsvrAddr        = 127.0.0.1:9529`
	cf := utils.HandleConfigFile(inifilename, defcfg)
	rconT = cf.MustInt("BASE", "rconT")
	jobsvrAddr = cf.MustValue("BASE", "jobsvrAddr")
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
	var agentServer utils.TFunc = rcsagent.StartRPCserver

	if e, tc = utils.NewTClient(jobsvrAddr, rconT, 0, true, agentServer); tc != nil {
		log.Fatalln(tc.Connect())
	}
	log.Fatalln(e)
}
