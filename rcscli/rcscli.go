package main

import (
	"bufio"
	"encoding/json"
	//	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	//"net"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	//	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"rcs/rcsagent"
	"rcs/utils"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type FileInfo struct {
	//ErrStatus   string
	Url, Md5str string
	Size        int64
}

const (
	sApiUrl                    = `http://127.0.0.1:9527/runtask`
	gettasksfnumsApiUrl        = `http://127.0.0.1:9528/gettasksfnums`
	gettaskresultAPiUrl        = `http://127.0.0.1:9528/gettaskresult`
	getagentresultApiUrl       = `http://127.0.0.1:9528/getAgentResult`
	getagentresultinsuccApiUrl = `http://127.0.0.1:9528/getagentresultinsucc`
	getagentresultinfailApiUrl = `http://127.0.0.1:9528/getagentresultinfail`
	TaskHandleTimeout          = 10
	fileregistry               = `http://127.0.0.1:8096/upload`
)

var logfile *os.File
var errs error
var suc, fad int32

func colorize(text string, status string) string {
	out := ""
	switch status {
	case "blue":
		out = "\033[32;1m" // Blue
	case "red":
		out = "\033[31;1m" // Red
	case "yell":
		out = "\033[33;1m" // Yellow
	case "green":
		out = "\033[34;1m" // Green
	default:
		out = "\033[0m" // Default
	}
	return out + text + "\033[0m"
}
func ReadlineAsSlice(fileName string) ([]string, error) {
	list := make([]string, 0)
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	//buf := bufio.NewReader(f)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		//	log.Println("line:", line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		/*if ip := net.ParseIP(line); ip == nil { //ip格式校验,正式版需加上
			return nil, errors.New("contain invalid ip form in file!")
		}*/
		list = append(list, line)
	}
	return list, scanner.Err()
}

type options struct {
	performance uint
	target      string
	targetfile  string
	file        string
	args        string
	dst         string
}

func (op options) createTaskReq() (*utils.RcsTaskReqJson, error) {
	rr := new(utils.RcsTaskReqJson)
	rr.Tp = uint8(op.performance)
	if op.target != "" {
		rr.Targets = strings.Split(op.target, ",")
	} else {
		ips, err := ReadlineAsSlice(op.targetfile)
		if err != nil {
			return nil, err
		}
		if ips == nil {
			return nil, errors.New("has no targets")
		}
		rr.Targets = ips
	}

	var err error
	var postfile_rsp *FileInfo

	if op.file != "" {
		err, postfile_rsp = postFile(op.file, fileregistry)
		if err != nil {
			return nil, err
		}
		if postfile_rsp.Url == "" || postfile_rsp.Md5str == "" {
			return nil, errors.New("something err when upload file to registry")
		}
	}
	switch rr.Tp {
	case rcsagent.ScriptExec:
		if op.file == "" {
			return nil, errors.New("No script file specify")
		}
		atreq := new(rcsagent.Script_Run_Req)
		//atreq.FileUrl = postfile_rsp.Url + `?rename=` + filepath.Base(op.file)
		atreq.FileUrl = postfile_rsp.Url
		atreq.FileMd5 = postfile_rsp.Md5str
		atreq.ScriptArgs = strings.Split(op.args, ",")
		rr.AtomicReq, _ = json.Marshal(atreq)
	case rcsagent.FilePush:
		if op.file == "" {
			return nil, errors.New("No file specify")
		}
		if op.dst == "" {
			return nil, errors.New("No dst specify")
		}
		if !filepath.IsAbs(op.dst) {
			return nil, errors.New("the -dst dir is not absolute")
		}
		d, f := filepath.Split(op.dst)
		atreq := new(rcsagent.File_Push_Req)
		if f == "" {
			atreq.FileUrl = postfile_rsp.Url
		} else {
			atreq.FileUrl = postfile_rsp.Url + `?rename=` + f
		}
		atreq.FileMd5 = postfile_rsp.Md5str
		atreq.DstPath = d
		rr.AtomicReq, _ = json.Marshal(atreq)

	case rcsagent.RcsAgentStop:
		atreq := new(rcsagent.Rcs_Stop_Req)
		rr.AtomicReq, _ = json.Marshal(atreq)
	case rcsagent.RcsAgentRestart:
		atreq := new(rcsagent.Rcs_Restart_Req)
		rr.AtomicReq, _ = json.Marshal(atreq)
	case rcsagent.RcsAgentUpgrade:
		atreq := new(rcsagent.Rcs_Upgrade_Req)
		rr.AtomicReq, _ = json.Marshal(atreq)
	case rcsagent.RcsAgentHeartBeat:
		atreq := new(rcsagent.Rcs_HeartBeat_Req)
		atreq.Msg = "Hello"
		rr.AtomicReq, _ = json.Marshal(atreq)
	default:
		return nil, errors.New("unkwon -p option !")
	}
	return rr, nil
}
func init() {
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
	t := flag.String("t", "", "-t=127.0.0.1,127.0.0.2... ,specify the targets")
	tf := flag.String("tf", "", "-tf=iplist.txt ,specify the file with targets,ignored when -t option is set")
	f := flag.String("f", "", "-f=/home/www/aaa.txt ,specify the local file which will be excuted on agents or distributed to agents")
	p := flag.Uint("p", 1, "-p=0 ,specify the operation,from 0 to 5,0 for run script,1 for distribute file")
	a := flag.String("args", "", "-args=arg1,arg2,arg3... ,specify the args when run script")
	d := flag.String("dst", "", "-dst=/var/www/ ,specify the absolute destination dir when distribute file")
	flag.Parse()
	if len(os.Args) < 4 {
		log.Fatalln("Params not enough,pls check!")
	}
	ops := options{}
	ops.performance = *p
	ops.target = *t
	ops.targetfile = *tf
	ops.file = *f
	ops.args = *a
	ops.dst = *d

	rr, err := ops.createTaskReq() //create task
	if err != nil {
		log.Fatalln(err)
	}

	vv, err := asyncSendTask(rr, sApiUrl) //submit task
	if err != nil {
		log.Fatalln(err)
	}

	uuid := vv.Uuid
	suc, fad = 0, 0
	an := len(rr.Targets)
	wg := new(sync.WaitGroup)
	wg.Add(an)

	for _, aip := range rr.Targets { //get result
		go getAgentResult(uuid, aip, wg)
	}
	wg.Wait()
	log.Println("-------------------------------------------------------------------------")
	log.Println("Task completed", uuid, time.Since(start).Nanoseconds()/1000000, "ms", suc, fad, int32(an)-suc-fad)
}

func asyncSendTask(rr *utils.RcsTaskReqJson, sApiUrl string) (*utils.MasterApiResp, error) {
	vv := new(utils.MasterApiResp)
	data, err := json.Marshal(rr)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("POST", sApiUrl, strings.NewReader(string(data)))
	req.Header.Set("Connection", "close")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(strconv.FormatInt(int64(resp.StatusCode), 10))
	}

	if err = json.NewDecoder(resp.Body).Decode(vv); err != nil {
		return nil, err
	}
	if err = resp.Body.Close(); err != nil {
		log.Println(err)
	}
	return vv, nil
}
func getAgentResult(uid, ip string, wg *sync.WaitGroup) {
	var (
		i    int
		resp *http.Response
		e    error
		vv   *utils.GetAgentResultFromRedisResp
	)
	defer wg.Done()
	vv = new(utils.GetAgentResultFromRedisResp)
	for i = 0; i < TaskHandleTimeout*10; i++ {
		time.Sleep(time.Second / 10)
		if er := queryAgentresultByapi(getagentresultinsuccApiUrl, uid, ip, resp, e, vv); er == nil {
			break
		}
		if er := queryAgentresultByapi(getagentresultinfailApiUrl, uid, ip, resp, e, vv); er == nil {
			break
		}
	}
	if i == TaskHandleTimeout*10 {
		log.Print(colorize("["+ip+"]", "yell")+"\n", "Time out"+"\n")
	}

}
func queryAgentresultByapi(apiurl, uid, ip string, resp *http.Response, e error, vv *utils.GetAgentResultFromRedisResp) error {
	req, _ := http.NewRequest("GET", apiurl+`?uuid=`+uid+`&ip=`+ip, nil)
	req.Header.Set("Connection", "close")
	req.Header.Set("Accept-Encoding", "gzip")
	resp, e = http.DefaultClient.Do(req)
	if e != nil || resp.StatusCode != 200 {
		//log.Println(ip+": ", e)
		return e
	}

	if e = json.NewDecoder(resp.Body).Decode(vv); e != nil {
		//log.Println(ip+": ", e)
		return e
	}
	if e = resp.Body.Close(); e != nil {
		//log.Println(ip+": ", e)
		return e
	}
	if vv.ErrStatus != "" { //结果为空是否也包含在此
		//log.Println(ip+": "+"api return err:", vv.ErrStatus)
		return errors.New(vv.ErrStatus)
	}
	switch apiurl {
	case getagentresultinsuccApiUrl:
		atomic.AddInt32(&suc, 1)
		log.Print(colorize("["+ip+"]", "blue")+"\n", vv.Res+"\n")
	case getagentresultinfailApiUrl:
		atomic.AddInt32(&fad, 1)
		log.Print(colorize("["+ip+"]", "red")+"\n", vv.Res+"\n")
	}
	return nil

}
func postFile(filename string, targetUrl string) (error, *FileInfo) { //上传文件逻辑：先get请求确认文件是否在对端存在(比对md5)，不存在才会真正执行post上传
	//log.Println("filename:", filename)
	//log.Print("targetUrl:", targetUrl)
	arsp := new(FileInfo)
	md, er := rcsagent.FileMd5(filename)
	if er != nil {
		log.Println(er)
		return er, nil
	}
	resp, err := http.Get(targetUrl + `?fmd5=` + md)
	if err != nil {
		log.Println(err)
		return err, nil
	}
	if resp.StatusCode != 200 {
		log.Println(errors.New(resp.Status))
		return errors.New(resp.Status), nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return err, nil
	}
	resp.Body.Close()
	err = json.Unmarshal(body, arsp)
	if err != nil {
		log.Println(err)
		return err, nil
	}
	if arsp.Md5str == md && arsp.Url != "" { //文件在fileregistry上已存在则不会上传,不存在则执行下面的上传动作
		//log.Println("file already exist in fileregistry")
		return nil, arsp
	} else {
		//log.Println("file doesn`t exist in fileregistry,uploading it...")
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
		if err != nil {
			log.Println(err)
			return err, nil
		}
		fh, err := os.Open(filename)
		if err != nil {
			log.Println(err)
			return err, nil
		}
		defer fh.Close()

		md5h := md5.New()
		defer md5h.Reset()
		_, err = io.Copy(io.MultiWriter(fileWriter, md5h), fh)
		if err != nil {
			log.Println(err)
			return err, nil
		}
		contentType := bodyWriter.FormDataContentType()
		bodyWriter.Close()
		fmd5 := hex.EncodeToString(md5h.Sum(nil))
		req, _ := http.NewRequest("POST", targetUrl+`?fmd5=`+fmd5, bodyBuf)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Connection", "close")
		resp, err = http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			log.Println(errors.New(err.Error() + resp.Status))
			return errors.New(err.Error() + resp.Status), nil
		}
		defer resp.Body.Close()
		resp_body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return err, nil
		}
		err = json.Unmarshal(resp_body, arsp)
		if err != nil {
			log.Println(err)
			return err, nil
		}
		return nil, arsp
	}
}
