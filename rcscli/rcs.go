package main

/* rcs command line tool
rcs [-t|-tf] cmd.script /tmp/test.sh "a b c"
rcs [-t|-tf] file.push  /tmp/test.txt "d:\a\b\c"
rcs [-t|-tf] process.stop "1.exe,2.exe,3.exe" true
*/
import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"rcs/rcsagent/modules"
	cli "rcs/rcscli/utils"
	"rcs/utils"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

var logfile *os.File
var errs error

func init() {
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	logfile, errs = os.OpenFile("log/rcscli.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	log.SetFlags(0)
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	//log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	defer logfile.Close()
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	start := time.Now()
	//----------------------
	t := flag.String("t", "", "-t 127.0.0.1,127.0.0.2... ,specify the targets")
	tf := flag.String("tf", "", "-tf iplist.txt ,specify the file with targets,ignored when -t option is set")
	flag.Parse()

	if len(os.Args) < 4 {
		log.Fatalln("Params not enough,pls check!")
	}
	if *t != "" && *tf != "" {
		log.Fatalln("-t and -tf can not be both specified,pls check!")
	}
	op := os.Args[3]
	targets := make([]string, 0)
	if *t != "" {
		targets = strings.Split(*t, ",")
	}
	if *tf != "" {
		ips, err := cli.ReadlineAsSlice(*tf)
		if err != nil {
			log.Fatalln(err)
		}
		if ips == nil {
			log.Fatalln("has no targets")
		}
		targets = ips
	}
	rr := new(utils.RcsTaskReqJson)
	rr.Tp = op
	rr.Targets = targets
	switch op {
	case "file.push":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			log.Println(`rcs [-t|-tf] ` + op + ` LocalFileName RemoteDst`)
			return
		}
		err, postfile_rsp := cli.PostFile(os.Args[4], cli.Fileregistry)
		if err != nil {
			log.Println(err)
			return
		}
		if postfile_rsp.Url == "" || postfile_rsp.Md5str == "" {
			log.Println("something err when upload file to registry")
			return
		}
		atomicReq := new(modules.File_push_req)
		atomicReq.Sfileurl = postfile_rsp.Url
		atomicReq.Sfilemd5 = postfile_rsp.Md5str
		atomicReq.DstPath = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "file.pull":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			log.Println(`rcs [-t|-tf] ` + op + ` RemoteFileName LocalDst`)
			return
		}
		atomicReq := new(modules.File_pull_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Dstdir = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.cp":
		if len(os.Args) < 7 {
			log.Println("Params not enough,pls check!")
			log.Println(`rcs [-t|-tf] ` + op + ` srcpath dstpath [true|false]`)
			return
		}
		atomicReq := new(modules.File_cp_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Dfilepath = os.Args[5]
		atomicReq.Wodir, _ = strconv.ParseBool(os.Args[6])
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.del":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			log.Println(`rcs [-t|-tf] ` + op + ` srcpath [true|false]`)
			return
		}
		atomicReq := new(modules.File_del_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Wobak, _ = strconv.ParseBool(os.Args[5])
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.grep":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.File_grep_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Patternstr = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.replace":
		if len(os.Args) < 7 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.File_replace_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Patternstr = os.Args[5]
		atomicReq.Repltext = os.Args[6]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.mreplace":
		if len(os.Args) < 8 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.File_mreplace_req)
		atomicReq.Sfiledir = os.Args[4]
		atomicReq.Filenamepatternstr = os.Args[5]
		atomicReq.Patternstr = os.Args[6]
		atomicReq.Repltext = os.Args[7]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.md5sum":
		if len(os.Args) < 5 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.File_md5sum_req)
		atomicReq.Sfilepath = os.Args[4]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.ckmd5sum":
		if len(os.Args) < 5 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.File_ckmd5sum_req)
		atomicReq.Md5filepath = os.Args[4]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "cmd.script":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		err, postfile_rsp := cli.PostFile(os.Args[4], cli.Fileregistry)
		if err != nil {
			log.Println(err)
			return
		}
		if postfile_rsp.Url == "" || postfile_rsp.Md5str == "" {
			log.Println("something err when upload file to registry")
			return
		}
		atomicReq := new(modules.Cmd_script_req)
		atomicReq.FileUrl = postfile_rsp.Url
		atomicReq.FileMd5 = postfile_rsp.Md5str
		atomicReq.ScriptArgs = strings.Split(os.Args[5], " ")
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "os.restart":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.Os_restart_req)
		atomicReq.Delay, _ = strconv.ParseBool(os.Args[4])
		atomicReq.Delaysecond, _ = strconv.ParseInt(os.Args[5], 10, 64)
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "os.shutdown":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.Os_shutdown_req)
		atomicReq.Delay, _ = strconv.ParseBool(os.Args[4])
		atomicReq.Delaysecond, _ = strconv.ParseInt(os.Args[5], 10, 64)
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "os.setpwd":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.Os_setpwd_req)
		atomicReq.Username = os.Args[4]
		atomicReq.Passwd = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "firewall.set":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.Firewall_set_req)
		atomicReq.Rulename = strings.Split(os.Args[4], ",")
		atomicReq.Op = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "process.stop":
		if len(os.Args) < 6 {
			log.Println("Params not enough,pls check!")
			return
		}
		atomicReq := new(modules.Process_stop_req)
		atomicReq.Imagename = strings.Split(os.Args[4], ",")
		atomicReq.Doforce, _ = strconv.ParseBool(os.Args[5])
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "rcs.ping":
		atomicReq := new(modules.Rcs_ping_req)
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	default:
		log.Fatalln("unknow command:" + op)
	}

	vv, err := cli.AsyncSendTask(rr, cli.SApiUrl) //submit task
	if err != nil {
		log.Fatalln(err)
	}

	uuid := vv.Uuid
	suc := new(int32)
	fad := new(int32)
	an := len(rr.Targets)
	wg := new(sync.WaitGroup)
	wg.Add(an)

	for _, aip := range rr.Targets { //get result
		go cli.GetAgentResult(uuid, aip, wg, suc, fad)
	}
	wg.Wait()
	log.Println("-------------------------------------------------------------------------")
	log.Println("Task completed", uuid, time.Since(start).Nanoseconds()/1000000, "ms", *suc, *fad, int32(an)-*suc-*fad)
}
