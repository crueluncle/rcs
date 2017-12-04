package rcsagent //定义共用对象及函数

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const ( //支持的原子操作类型
	ScriptExec        uint8 = iota //脚本执行
	FilePush                       //文件分发
	RcsAgentRestart                //agent重启
	RcsAgentUpgrade                //agent升级
	RcsAgentStop                   //agent退出
	RcsAgentHeartBeat              //agent心跳
)

//-----------------------------------------------
type RpcCallRequest interface { //表示一个原子请求
	Handle(*RpcCallResponse) error
	GetFileUrl() string
	GetFileMd5() string
	SetFileUrl(string)
}
type RpcCallResponse struct { //表示一个原子请求的响应
	Flag   bool
	Result string
}

//以下定义6种原子请求类,原子操作基本固定,不会增删
type Script_Run_Req struct {
	FileUrl, FileMd5 string
	ScriptArgs       []string
}
type File_Push_Req struct {
	FileUrl, FileMd5 string
	DstPath          string
}
type Rcs_Restart_Req struct {
}
type Rcs_Upgrade_Req struct {
}
type Rcs_Stop_Req struct {
}
type Rcs_HeartBeat_Req struct {
	Msg string
}

func Downloadfilefromurl(srcfileurl, srcfilemd5, dstdir string) error {
	//目标文件名与url中uri一致，若文件存在且md5一致则不会下载
	//	log.Println("srcfileurl:", srcfileurl, "dstdir:", dstdir)
	u, e := url.Parse(srcfileurl)
	if e != nil {
		return e
	}
	//bn := strings.Split(u.RequestURI(), `/`)
	filename := u.Query().Get("rename")
	if filename == "" {
		filename = filepath.Base(u.RequestURI())
		if filename == "" {
			return errors.New("srcfileurl is invalid:" + srcfileurl)
		}
	}
	dstfilepath := filepath.Join(dstdir, filename)
	//log.Println("dstfilepath:", dstfilepath)
	if Isexist(dstfilepath) {
		md, err := FileMd5(dstfilepath)
		if err == nil && md == srcfilemd5 {
			return nil
		}
	}
	req, _ := http.NewRequest("GET", strings.Split(srcfileurl, `?`)[0], nil)
	//req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Connection", "close")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		//log.Println(err)
		return err
	}
	if resp.StatusCode != 200 {
		//log.Println(errors.New(resp.Status))
		return errors.New(resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(dstfilepath), 0777); err != nil {
		return err
	}
	f1, e := os.OpenFile(dstfilepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if e != nil {
		return e
	}
	md5h := md5.New()
	_, err = io.Copy(io.MultiWriter(f1, md5h), resp.Body)
	if err != nil {
		return err
	}
	if err = f1.Close(); err != nil {
		return err
	}
	if err = resp.Body.Close(); err != nil {
		return err
	}
	if hex.EncodeToString(md5h.Sum(nil)) == srcfilemd5 {
		return nil
	} else {
		return errors.New("md5sum not matched")
	}
}

func FileMd5(filepath string) (string, error) {
	file, inerr := os.Open(filepath)
	defer file.Close()
	if inerr == nil {
		md5h := md5.New()
		if _, err := io.Copy(md5h, file); err != nil {
			return "", err
		}
		chksum := hex.EncodeToString(md5h.Sum(nil))
		return chksum, nil
	}
	return "", inerr
}
func Isexist(path string) bool {
	_, err := os.Lstat(path)
	if err != nil && !strings.Contains(err.Error(), "it is being used") {
		return false
	}
	return true
}
