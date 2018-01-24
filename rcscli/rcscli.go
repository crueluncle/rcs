package main

//commandline tool
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
var successipfilename, failipfilename, timeoutipfilename string

func init() {
	tm := strconv.FormatInt(time.Now().Unix(), 10)

	if err := os.MkdirAll(`result`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	successipfilename = `result/success.ip_` + tm
	failipfilename = `result/fail.ip_` + tm
	timeoutipfilename = `result/timeout.ip_` + tm
	logfile, errs = os.OpenFile("log/rcscli.log", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	cli.Success, errs = os.OpenFile(successipfilename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	cli.Fail, errs = os.OpenFile(failipfilename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	cli.Timeout, errs = os.OpenFile(timeoutipfilename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	log.SetFlags(0)
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	//log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
SApiUrl                    = http://127.0.0.1:9527/runtask
GettasksfnumsApiUrl        = http://127.0.0.1:9528/gettasksfnums
GettaskresultAPiUrl        = http://127.0.0.1:9528/gettaskresult
getagentresultApiUrl       = http://127.0.0.1:9528/getAgentResult
getagentresultinsuccApiUrl = http://127.0.0.1:9528/getagentresultinsucc
getagentresultinfailApiUrl = http://127.0.0.1:9528/getagentresultinfail
TaskHandleTimeout          = 600                           
Fileregistry               = http://127.0.0.1:8096/upload`

	cf := utils.HandleConfigFile("cfg/rcscli.ini", defcfg)
	cli.SApiUrl = cf.MustValue("BASE", "SApiUrl")
	cli.GettasksfnumsApiUrl = cf.MustValue("BASE", "GettasksfnumsApiUrl")
	cli.GettaskresultAPiUrl = cf.MustValue("BASE", "GettaskresultAPiUrl")
	cli.GetagentresultApiUrl = cf.MustValue("BASE", "getagentresultApiUrl")
	cli.GetagentresultinsuccApiUrl = cf.MustValue("BASE", "getagentresultinsuccApiUrl")
	cli.GetagentresultinfailApiUrl = cf.MustValue("BASE", "getagentresultinfailApiUrl")
	cli.TaskHandleTimeout = cf.MustInt("BASE", "TaskHandleTimeout")
	cli.Fileregistry = cf.MustValue("BASE", "Fileregistry")
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	defer logfile.Close()
	defer cli.Success.Close()
	defer cli.Fail.Close()
	defer cli.Timeout.Close()
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	start := time.Now()
	//----------------------
	t := flag.String("t", "", "-t 127.0.0.1,127.0.0.2... ,specify the targets")
	tf := flag.String("tf", "", "-tf iplist.txt ,specify the file with targets,ignored when -t option is set")
	flag.Parse()

	if len(os.Args) < 4 {
		log.Fatalln("Usage: " + os.Args[0] + `[-t|-tf] <targets|targetsfile> <module.function>
-----------------------------------------------------------------------------------------------------------------
module.function belows:
file.push          -- for push local file to remote targets
file.pull          -- for pull remote file on targets to localhost
file.cp            -- run 'copy' command on remote targets for copy file or directory
file.del           -- run 'delete' command on remote targets for delete file or directory
file.rename        -- run 'rename' command on remote targets for rename file or directory
file.grep          -- run 'grep' command on remote targets for grep file content
file.replace       -- run 'replace' command on remote targets for replace text file
file.mreplace      -- run 'mreplace' command on remote targets for replace multiple text files
file.md5sum        -- run 'md5sum' command on remote targets for compute file md5sum 
file.ckmd5sum      -- run 'ckmd5sum' command on remote targets for check md5sum of files defined in md5sum file
file.zip           -- run 'zip' command on remote targets for zip files or directory
file.unzip         -- run 'unzip' command on remote targets for unzip zipdfile
cmd.run            -- exec command on remote targets
cmd.script         -- exec local scriptfile on remote targets,push to remote and run
os.restart         -- restart remote targets
os.shutdown        -- shutdown remote targets
os.setpwd          -- set user password on remote targets
firewall.setrules  -- set filewall rules on remote targets
process.stop       -- stop the specified process on remote targets
rcs.ping           -- for rcsping remote targets`)
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
			//log.Println("Params not enough,pls check!")
			log.Println("Usage: " + op + ` <LocalFileName>  <RemoteDst>`)
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
			log.Println("Usage: " + op + ` <RemoteFileName>  <LocalDir>`)
			return
		}
		atomicReq := new(modules.File_pull_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Dstdir = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.cp":
		if len(os.Args) < 6 {
			log.Println("Usage: " + op + ` <srcpath>  <dstpath> [true|false]`)
			return
		}
		atomicReq := new(modules.File_cp_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Dfilepath = os.Args[5]
		if len(os.Args) > 6 {
			atomicReq.Wodir, _ = strconv.ParseBool(os.Args[6])
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.del":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <srcpath>  [true|false]`)
			return
		}
		atomicReq := new(modules.File_del_req)
		atomicReq.Sfilepath = os.Args[4]
		if len(os.Args) > 5 {
			atomicReq.Wobak, _ = strconv.ParseBool(os.Args[5])
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "file.rename":
		if len(os.Args) < 6 {
			log.Println("Usage: " + op + ` <srcpath>  <newname>`)
			return
		}
		atomicReq := new(modules.File_rename_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Newname = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "file.grep":
		if len(os.Args) < 6 {
			log.Println("Usage: " + op + ` <srcfilepath>  <patternstr>`)
			return
		}
		atomicReq := new(modules.File_grep_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Patternstr = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.replace":
		if len(os.Args) < 7 {
			log.Println("Usage: " + op + ` <srcfilepath>  <patternstr> <repltext>`)
			return
		}
		atomicReq := new(modules.File_replace_req)
		atomicReq.Sfilepath = os.Args[4]
		atomicReq.Patternstr = os.Args[5]
		atomicReq.Repltext = os.Args[6]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.mreplace":
		if len(os.Args) < 8 {
			log.Println("Usage: " + op + ` <srcfilepath>  <filenamepatternstr> <patternstr> <repltext>`)
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
			log.Println("Usage: " + op + ` <filepath>`)
			return
		}
		atomicReq := new(modules.File_md5sum_req)
		atomicReq.Sfilepath = os.Args[4]
		rr.AtomicReq, _ = json.Marshal(atomicReq)

	case "file.ckmd5sum":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <Md5filepath>`)
			return
		}
		atomicReq := new(modules.File_ckmd5sum_req)
		atomicReq.Md5filepath = os.Args[4]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "file.zip":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <srcfilepath>  [Zipfilepath]`)
			return
		}
		atomicReq := new(modules.File_zip_req)
		atomicReq.Sfilepath = os.Args[4]
		if len(os.Args) > 5 {
			atomicReq.Zipfilepath = os.Args[5]
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "file.unzip":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <Zipfilepath>  [Dstdir]  [Wdir]`)
			return
		}
		atomicReq := new(modules.File_unzip_req)
		atomicReq.Zipfilepath = os.Args[4]
		if len(os.Args) > 5 {
			atomicReq.Dstdir = os.Args[5]
		}
		if len(os.Args) > 6 {
			atomicReq.Wdir, _ = strconv.ParseBool(os.Args[6])
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "cmd.run":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <command>  [CmdArgs...]`)
			return
		}
		atomicReq := new(modules.Cmd_run_req)
		atomicReq.Cmd = os.Args[4]
		if len(os.Args) > 5 {
			atomicReq.CmdArgs = os.Args[5:]
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "cmd.script":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <scriptfilepath>  [ScriptArgs] [shell=Stype]`)
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
		if len(os.Args) == 6 {
			if strings.HasPrefix(os.Args[5], `shell=`) {
				atomicReq.Stype = os.Args[5]
			} else {
				atomicReq.ScriptArgs = strings.Split(os.Args[5], " ")
			}
		}
		if len(os.Args) > 6 {
			atomicReq.ScriptArgs = strings.Split(os.Args[5], " ")
			atomicReq.Stype = os.Args[6]
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "os.restart":
		//log.Println("Usage: " + op + ` [Delay]  [Delaysecond]`)
		atomicReq := new(modules.Os_restart_req)
		if len(os.Args) > 4 {
			atomicReq.Delay, _ = strconv.ParseBool(os.Args[4])
		}
		if len(os.Args) > 5 {
			atomicReq.Delaysecond, _ = strconv.ParseInt(os.Args[5], 10, 64)
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "os.shutdown":
		//log.Println("Usage: " + op + ` [Delay]  [Delaysecond]`)
		atomicReq := new(modules.Os_shutdown_req)
		if len(os.Args) > 4 {
			atomicReq.Delay, _ = strconv.ParseBool(os.Args[4])
		}
		if len(os.Args) > 5 {
			atomicReq.Delaysecond, _ = strconv.ParseInt(os.Args[5], 10, 64)
		}
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "os.setpwd":
		if len(os.Args) < 6 {
			log.Println("Usage: " + op + ` <Username>  <Password>`)
			return
		}
		atomicReq := new(modules.Os_setpwd_req)
		atomicReq.Username = os.Args[4]
		atomicReq.Passwd = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "firewall.set":
		if len(os.Args) < 6 {
			log.Println("Usage: " + op + ` <Rulename>  <disable|enable|del>`)
			return
		}
		atomicReq := new(modules.Firewall_set_req)
		atomicReq.Rulename = strings.Split(os.Args[4], ",")
		atomicReq.Op = os.Args[5]
		rr.AtomicReq, _ = json.Marshal(atomicReq)
	case "process.stop":
		if len(os.Args) < 5 {
			log.Println("Usage: " + op + ` <Imagename>  [Doforce]`)
			return
		}
		atomicReq := new(modules.Process_stop_req)
		atomicReq.Imagename = strings.Split(os.Args[4], ",")
		if len(os.Args) > 5 {
			atomicReq.Doforce, _ = strconv.ParseBool(os.Args[5])
		}
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
