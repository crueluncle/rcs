package utils

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	//"net"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"mime/multipart"
	"net/http"
	"os"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	//"github.com/qiniu/iconv"
)

const (
	SApiUrl                    = `http://127.0.0.1:9527/runtask`
	GettasksfnumsApiUrl        = `http://127.0.0.1:9528/gettasksfnums`
	GettaskresultAPiUrl        = `http://127.0.0.1:9528/gettaskresult`
	getagentresultApiUrl       = `http://127.0.0.1:9528/getAgentResult`
	getagentresultinsuccApiUrl = `http://127.0.0.1:9528/getagentresultinsucc`
	getagentresultinfailApiUrl = `http://127.0.0.1:9528/getagentresultinfail`
	TaskHandleTimeout          = 10                             //一个任务task执行默认超时时间
	Fileregistry               = `http://127.0.0.1:8096/upload` //文件仓库上传地址
)

type FileInfo struct { //api返回给调用者的消息
	Url, Md5str string
	Size        int64
}

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

func AsyncSendTask(rr *utils.RcsTaskReqJson, sApiUrl string) (*utils.MasterApiResp, error) {
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
func GetAgentResult(uid, ip string, wg *sync.WaitGroup, suc, fad *int32) {
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
		if er := queryAgentresultByapi(getagentresultinsuccApiUrl, uid, ip, resp, e, vv, suc, fad); er == nil {
			break
		}
		if er := queryAgentresultByapi(getagentresultinfailApiUrl, uid, ip, resp, e, vv, suc, fad); er == nil {
			break
		}
	}
	if i == TaskHandleTimeout*10 {
		log.Print(colorize("["+ip+"]", "yell")+"\n", "Time out"+"\n")
	}

}
func queryAgentresultByapi(apiurl, uid, ip string, resp *http.Response, e error, vv *utils.GetAgentResultFromRedisResp, suc, fad *int32) error {
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
		atomic.AddInt32(suc, 1)
		log.Print(colorize("["+ip+"]", "blue")+"\n", vv.Res+"\n")
	case getagentresultinfailApiUrl:
		atomic.AddInt32(fad, 1)
		log.Print(colorize("["+ip+"]", "red")+"\n", vv.Res+"\n")
	}
	return nil

}
func PostFile(filename string, targetUrl string) (error, *FileInfo) { //上传文件逻辑：先get请求确认文件是否在对端存在(比对md5)，不存在才会真正执行post上传
	//log.Println("filename:", filename)
	//log.Print("targetUrl:", targetUrl)
	arsp := new(FileInfo)
	md, er := modules.FileMd5(filename)
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
