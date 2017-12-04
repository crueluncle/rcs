package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"rcs/rcsagent"
	"rcs/utils"
	"runtime/debug"
	"syscall"

	"github.com/kardianos/service"
)

var (
	rconT      int
	jobsvrAddr string
)
var logf *os.File
var logger service.Logger

type program struct{}

func (p *program) Start(s service.Service) error {
	if err := s.Install(); err != nil {
		log.Println(err)
	}
	go p.run()
	return nil
}
func (p *program) Stop(s service.Service) error {
	return nil
}
func (p *program) run() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()

	var e error
	var tc *utils.TClient
	var agentServer utils.TFunc = rcsagent.StartRPCserver

	if e, tc = utils.NewTClient(jobsvrAddr, rconT, 0, true, agentServer); tc != nil {
		log.Fatalln(tc.Connect())
	}
	log.Fatalln(e)
}
func init() {
	gob.Register(&rcsagent.Script_Run_Req{})
	gob.Register(&rcsagent.File_Push_Req{})
	gob.Register(&rcsagent.Rcs_Restart_Req{})
	gob.Register(&rcsagent.Rcs_Stop_Req{})
	gob.Register(&rcsagent.Rcs_Upgrade_Req{})
	gob.Register(&rcsagent.Rcs_HeartBeat_Req{})
	file, _ := exec.LookPath(os.Args[0])
	//path, _ := os.Getwd()
	path := filepath.Dir(file)
	logfilename := filepath.Join(path, `log/rcsagent.log`)
	logf, _ = os.OpenFile(logfilename, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0777)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(logf)
	//log.SetOutput(io.MultiWriter(logf, os.Stdout))
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

	svcConfig := &service.Config{
		Name:        "rcsagent",
		DisplayName: "rcsagent",
		Description: "rcsagent service.",
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) == 2 {
		if os.Args[1] == "start" {
			err = service.Control(s, "install")
			if err != nil {
				log.Println(err)
			}
			err = service.Control(s, "start")
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err = service.Control(s, os.Args[1])
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		fmt.Println("Rekagent is running in foreground now,Typing command 'rcsagent.exe start' for run in background is recommended!")
		err = s.Run()
		if err != nil {
			log.Println(err)
			logger.Error(err)
		}
	}
	select {}
}
