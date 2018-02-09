package main

import (
	"io"
	"log"
	"os"
	"rcs/rcsjobsvr/rcsjobfilecache/modules"
	"rcs/utils"
	"runtime/debug"
)

var (
	filecacheAddr,
	filecachedir string //文件缓存服务器根目录
)
var logfile *os.File

func init() { //初始化操作
	//处理日志
	var err error
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	logfile, err = os.OpenFile("log/jobfilecache.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}

	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)

	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
filecacheAddr      = 0.0.0.0:9530
filecachedir = JobsrvFileCacheDir`

	cf := utils.HandleConfigFile("cfg/jobfilecache.ini", defcfg)
	filecacheAddr = cf.MustValue("BASE", "filecacheAddr")
	filecachedir = cf.MustValue("BASE", "filecachedir")

}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
		}
	}()
	defer logfile.Close()

	myfileServer := modules.NewFileSvr(filecacheAddr, filecachedir)

	myfileServer.ServeFile()

}
