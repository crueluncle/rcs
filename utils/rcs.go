package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rcs/rcsagent/modules"
	"time"

	"github.com/Unknwon/goconfig"
)

const (
	Version   = "5.0"
	BuildTime = "2017-12-1"
	Author    = "careyzhang"
)

type RcsTaskReqJson struct {
	Runid     string
	Targets   []string
	Tp        string
	AtomicReq json.RawMessage
}
type RcsTaskReq struct {
	Runid   string
	Targets []string
	modules.Atomicrequest
}
type RcsTaskResp struct { /*jobsvr返回给master的响应结构,存储到redis-hash表中
	对于每一个执行态runid,生存2个hash表：runid:true存放flag为true的RcsResponse对象：hset 1000:true 1.1.1.1 result(为resutl字段的json串)
	runid:false存放flag为false的RcsResponse对象;调用侧获取某个runid
	成功失败的数量：hlen runid:true/hlen runid:false
	获取成功/失败ip:hkeys runid:true/hkeys runid:false
	*/
	Runid   string //执行态id,全局唯一
	AgentIP string
	modules.Atomicresponse
}

func (task *RcsTaskReqJson) Parse() (*RcsTaskReq, error) {
	var atomicReq modules.Atomicrequest
	var taskreq = new(RcsTaskReq)
	switch task.Tp {
	case "file.push":
		atomicReq = new(modules.File_push_req)
	case "file.pull":
		atomicReq = new(modules.File_pull_req)
	case "file.cp":
		atomicReq = new(modules.File_cp_req)
	case "file.del":
		atomicReq = new(modules.File_del_req)
	case "file.grep":
		atomicReq = new(modules.File_grep_req)
	case "file.replace":
		atomicReq = new(modules.File_replace_req)
	case "file.mreplace":
		atomicReq = new(modules.File_mreplace_req)
	case "file.md5sum":
		atomicReq = new(modules.File_md5sum_req)
	case "file.ckmd5sum":
		atomicReq = new(modules.File_ckmd5sum_req)
	case "file.zip":
		atomicReq = new(modules.File_zip_req)
	case "file.unzip":
		atomicReq = new(modules.File_unzip_req)
	case "cmd.script":
		atomicReq = new(modules.Cmd_script_req)
	case "os.restart":
		atomicReq = new(modules.Os_restart_req)
	case "os.shutdown":
		atomicReq = new(modules.Os_shutdown_req)
	case "os.setpwd":
		atomicReq = new(modules.Os_setpwd_req)
	case "firewall.set":
		atomicReq = new(modules.Firewall_set_req)
	case "process.stop":
		atomicReq = new(modules.Process_stop_req)
	case "rcs.ping":
		atomicReq = new(modules.Rcs_ping_req)
	default:
		return nil, errors.New("unknow command:" + task.Tp)
	}
	if err := json.Unmarshal(task.AtomicReq, atomicReq); err != nil {
		return nil, err
	}
	taskreq.Runid = task.Runid
	taskreq.Targets = task.Targets
	taskreq.Atomicrequest = atomicReq
	return taskreq, nil
}

type MasterApiResp struct { //masterapi返回给api调用者的消息
	ErrStatus string
	Uuid      string
}
type KeepaliveMsg struct { //mater与jobsvr之间的探测消息
	Id string
	Sn int
}

func (rs *MasterApiResp) EncodeJson(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	jsonenc := json.NewEncoder(w)
	if e := jsonenc.Encode(rs); e != nil {
		log.Println(e)
	}
	return
}
func Md5sum(message []byte) ([]byte, error) {
	md := md5.New()
	_, err := md.Write(message)
	chksum := []byte(hex.EncodeToString(md.Sum(nil)))
	return chksum, err
}

func FileSizeAndMd5(filepath string) (int64, string, error) {
	file, inerr := os.Open(filepath)
	defer file.Close()
	if inerr == nil {
		md5h := md5.New()
		sz, err := io.Copy(md5h, file)
		if err != nil {
			return 0, "", err
		}
		chksum := hex.EncodeToString(md5h.Sum(nil))
		return sz, chksum, nil
	}
	return 0, "", inerr
}
func Listfiles(dir string) []string {
	filelist := make([]string, 0)
	wf := func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		filelist = append(filelist, path)
		return nil
	}
	err := filepath.Walk(dir, wf)
	if err != nil {
		log.Printf("filepath.Walk() returned %v\n", err)
	}
	return filelist
}

func HandleConfigFile(configfilename, defaultconfig string) *goconfig.ConfigFile {
	if ex, _, _ := modules.Isexistdir(configfilename); !ex {
		log.Println("No cfg file exist,create it...")
		cfgfile, err := os.OpenFile(configfilename, os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			log.Fatal(err)
		}
		defer cfgfile.Close()
		_, err = cfgfile.WriteString(defaultconfig)
		if err != nil {
			log.Fatal("create cfg file failed!")
		}
		log.Println("Create cfg file success,pls edit the config file properly and then restart this program!")
		for i := 5; i > 0; i-- {
			log.Printf("Program will exit in %d seconds\n", i)
			time.Sleep(time.Second)
		}
		os.Exit(0)
	}
	cf, err := goconfig.LoadConfigFile(configfilename)
	if err != nil {
		log.Fatalln("LoadConfigFile:", err)
	}
	return cf
}
