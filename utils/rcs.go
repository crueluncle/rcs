package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rcs/rcsagent"
	"time"

	"github.com/Unknwon/goconfig"
)

const (
	Version   = "4.0"
	BuildTime = "2017-9-12"
	Author    = "careyzhang"
)

type RcsTaskReqJson struct {
	Runid     string
	Targets   []string
	Tp        uint8
	AtomicReq json.RawMessage
}
type RcsTaskReq struct {
	Runid     string
	Targets   []string
	AtomicReq rcsagent.RpcCallRequest
}
type RcsTaskResp struct {
	Runid   string
	AgentIP string
	rcsagent.RpcCallResponse
}

func (task *RcsTaskReqJson) Parse() *RcsTaskReq {
	var atomicReq rcsagent.RpcCallRequest
	var taskreq = new(RcsTaskReq)
	switch task.Tp {
	case rcsagent.ScriptExec:
		atomicReq = new(rcsagent.Script_Run_Req)
	case rcsagent.FilePush:
		atomicReq = new(rcsagent.File_Push_Req)
	case rcsagent.RcsAgentRestart:
		atomicReq = new(rcsagent.Rcs_Restart_Req)
	case rcsagent.RcsAgentStop:
		atomicReq = new(rcsagent.Rcs_Stop_Req)
	case rcsagent.RcsAgentUpgrade:
		atomicReq = new(rcsagent.Rcs_Upgrade_Req)
	case rcsagent.RcsAgentHeartBeat:
		atomicReq = new(rcsagent.Rcs_HeartBeat_Req)
	default:
		return nil
	}
	if err := json.Unmarshal(task.AtomicReq, atomicReq); err != nil {
		return nil
	}
	taskreq.Runid = task.Runid
	taskreq.Targets = task.Targets
	taskreq.AtomicReq = atomicReq
	return taskreq
}

type MasterApiResp struct {
	ErrStatus string
	Uuid      string
}
type KeepaliveMsg struct {
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
	if !rcsagent.Isexist(configfilename) {
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
