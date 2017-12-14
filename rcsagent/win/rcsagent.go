package main

//创建一个services，业务逻辑代码放到run()函数中
//可通过test.exe install/uninstall来安装/卸载服务，安装后可通过 start stop restart命令来控制
//直接双击则跑在前端
//日志仅写到文件中
import (
	//	"encoding/gob"
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

var (
	rconT      int    //agent断开jobsvr连接后，在多长的随机时间内重连jobsvr,agent数量可能较多，随机重连避免风暴
	jobsvrAddr string // jobsvr地址
)
var logf *os.File         //将start/stop/run中逻辑代码的日志记录到文件
var logger service.Logger //服务的系统日志器(将日志写到windows系统日志中，可在eventviewer中查看，不输出console)

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
	var agentServe utils.TFunc = modules.InitRPCserver_win

	if e, tc = utils.NewTClient(jobsvrAddr, rconT, 0, true, agentServe); tc != nil {
		log.Fatalln(tc.Connect())
	}
	log.Fatalln(e)
}
func init() {
	/*	gob.Register(&rcsagent.Script_Run_Req{})
		gob.Register(&rcsagent.File_Push_Req{})
		gob.Register(&rcsagent.Rcs_Restart_Req{})
		gob.Register(&rcsagent.Rcs_Stop_Req{})
		gob.Register(&rcsagent.Rcs_Upgrade_Req{})
		gob.Register(&rcsagent.Rcs_HeartBeat_Req{})
	*/
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
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(logf)
	log.SetOutput(io.MultiWriter(logf, os.Stdout))
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
	//在此处将标准log的输出定位到一个文件，应每次执行test.exe [cmd]时会重新打开文件，文件指针会重新指向文件开头，因此为保持日志连续性，在调用log的函数中需seek到文件末尾或者以追加的方式打开
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
