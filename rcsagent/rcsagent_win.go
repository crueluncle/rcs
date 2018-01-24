package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"runtime/debug"
	"syscall"

	"github.com/kardianos/service"
)

var logf *os.File
var logger service.Logger
var (
	RconT      int
	JobsvrAddr string
)

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
	var agentServe utils.TFunc = modules.InitRPCserver

	if e, tc = utils.NewTClient(JobsvrAddr, RconT, 0, true, agentServe); tc != nil {
		log.Fatalln(tc.Connect())
	}
	log.Fatalln(e)
}
func init() {
	gob.Register(&modules.File_push_req{})
	gob.Register(&modules.File_pull_req{})
	gob.Register(&modules.File_cp_req{})
	gob.Register(&modules.File_del_req{})
	gob.Register(&modules.File_rename_req{})
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
	if err := os.MkdirAll(filepath.Join(path, `log`), 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(filepath.Join(path, `cfg`), 0666); err != nil {
		log.Fatalln(err)
	}
	logfilename := filepath.Join(path, `log/rcsagent.log`)
	logf, _ = os.OpenFile(logfilename, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0777)
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(logf)
	log.SetOutput(io.MultiWriter(logf, os.Stdout))
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)

	inifilename := filepath.Join(path, `cfg/rcsagent.ini`)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
	[BASE]
	rconT             = 10
	jobsvrAddr        = 127.0.0.1:9529`
	cf := utils.HandleConfigFile(inifilename, defcfg)
	RconT = cf.MustInt("BASE", "rconT")
	JobsvrAddr = cf.MustValue("BASE", "jobsvrAddr")
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
	s, err := service.New(prg, svcConfig) //生成service 实例，之后通过service.Control进行控制
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) == 2 { //如果有参数则执行相关参数命令：install uninstall start stop restart
		if os.Args[1] == "start" { //先安装service再启动
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
	} else { //不带参数运行(比如直接双击)则跑在前端
		fmt.Println("Rekagent is running in foreground now,Typing command 'rcsagent.exe start' for run in background is recommended!")
		err = s.Run()
		if err != nil {
			log.Println(err)
			logger.Error(err)
		}
	}
	select {}
}
