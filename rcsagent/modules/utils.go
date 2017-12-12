package modules //定义共用对象及函数
//定义rpc服务模块
import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
type Atomicresponse struct { //表示一个原子请求的响应
	Flag   bool
	Result string
}

//以下定义6种原子请求类,原子操作基本固定
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
	if ex, dr, _ := Isexistdir(dstfilepath); ex && !dr {
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
func Cpall(sfilepath, dfilepath string, withoutdir bool) error { //文件、是目录则递归、存在则默认覆盖
	/*
		1.sfilepath是文件则将dfilepath视为文件
		2.sfilepath是目录,dfilepath视为目录，且默认带目录名及目录下所有内容cp到dfilepath目录下 sfilepath--->dfilepath/sfilepath

	*/
	ex, dr, err := Isexistdir(sfilepath)
	if !ex {
		return err
	}
	if !dr {
		return cpfile(sfilepath, dfilepath)
	} else { //是目录，递归cpfile
		wf := func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}
			if withoutdir { //不带目录名复制
				return cpfile(path, filepath.Join(dfilepath, strings.TrimPrefix(path, sfilepath)))
			} else { //带目录名复制
				return cpfile(path, filepath.Join(dfilepath, strings.TrimPrefix(path, filepath.Clean(sfilepath+`/../`))))
			}
		}
		if err := filepath.Walk(sfilepath, wf); err != nil {
			return err
		}
	}
	return nil
}
func cpfile(sfilepath, dfilepath string) error {
	if err := os.MkdirAll(filepath.Dir(dfilepath), 0777); err != nil {
		return err
	}
	sFile, err := os.Open(sfilepath)
	if err != nil {
		return err
	}
	defer sFile.Close()
	eFile, err := os.Create(dfilepath)
	if err != nil {
		return err
	}
	defer eFile.Close()
	_, err = io.Copy(eFile, sFile)
	if err != nil {
		return err
	}
	err = eFile.Sync()
	eFile.Sync()
	if err != nil {
		return err
	}
	return nil
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
func Isexistdir(name string) (isexist, isdir bool, err error) { //是否存在,是否为目录
	fi, err := os.Stat(name)
	if err == nil || os.IsExist(err) {
		isexist = true
		isdir = fi.IsDir()
		return isexist, isdir, err
	}
	if os.IsNotExist(err) {
		return false, false, err
	}
	isexist = true
	isdir = fi.IsDir()
	return isexist, isdir, err
}
func Listmatchfiles(dirname string, filenamepattern string) (error, []string) { //列出给定目录下，文件名匹配filenamepattern的所有文件
	ex, dr, err := Isexistdir(dirname)
	if !ex {
		return err, nil
	}
	if !dr {
		return errors.New(dirname + " is not a dir"), nil
	}
	filelist := []string{}
	wf := func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if rs, _ := regexp.MatchString(filenamepattern, f.Name()); rs {
			filelist = append(filelist, path)
		}
		return nil
	}
	err = filepath.Walk(dirname, wf)
	if err != nil {
		return err, nil
	}
	return nil, filelist
}
