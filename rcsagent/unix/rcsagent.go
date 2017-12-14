package main

//日志写到文件中
import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"rcs/rcsagent"
	"rcs/utils"
	"runtime/debug"
	"syscall"

	//"github.com/Unknwon/goconfig"
)

var (
	rconT      int    //agent断开jobsvr连接后，在多长的随机时间内重连jobsvr,agent数量可能较多，随机重连避免风暴
	jobsvrAddr string // jobsvr地址
)
var logf *os.File //将start/stop/run中逻辑代码的日志记录到文件

func init() {
	/*gob.Register(&rcsagent.Script_Run_Req{})
	gob.Register(&rcsagent.File_Push_Req{})
	gob.Register(&rcsagent.Rcs_Restart_Req{})
	gob.Register(&rcsagent.Rcs_Stop_Req{})
	gob.Register(&rcsagent.Rcs_Upgrade_Req{})
	gob.Register(&rcsagent.Rcs_HeartBeat_Req{})*/
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
	//在此处将标准log的输出定位到一个文件，应每次执行test.exe [cmd]时会重新打开文件，文件指针会重新指向文件开头，因此为保持日志连续性，在调用log的函数中需seek到文件末尾或者以追加的方式打开
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()
	defer logf.Close()

	var e error
	var tc *utils.TClient
	var agentServe utils.TFunc = rcsagent.InitRPCserver

	if e, tc = utils.NewTClient(jobsvrAddr, rconT, 0, true, agentServe); tc != nil {
		log.Fatalln(tc.Connect())
	}
	log.Fatalln(e)
}
