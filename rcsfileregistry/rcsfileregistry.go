//1.接收文件上传,返回url,size,md5,文件存储逻辑：basedir下自动生成用户名(如果有)+日期的目录
//2.提供文件下载,传输采用gzip压缩
//3.解决了不同用户同一天上传的同名文件相互覆盖问题;上传请求中带认证头,服务端获取用户名生存目录
//bugs:1.运行时,files目录及文件不能手动删除,若要删除则需要重启fileregistry服务来重新建立内存对象

package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rcs/utils"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const basedir string = `files`

var (
	upaddr, downaddr string
)
var FIlock = new(sync.Mutex)

type FileInfo struct { //api返回给调用者的消息
	Url, Md5str string
	Size        int64
}

var FileReg map[string]*FileInfo

func fileadd(md5 string, ar *FileInfo) {
	FIlock.Lock()
	defer FIlock.Unlock()
	FileReg[md5] = ar
}
func filedel(md5 string) {
	FIlock.Lock()
	defer FIlock.Unlock()
	delete(FileReg, md5)
}
func getfilei(md5 string) *FileInfo {
	return FileReg[md5]
}
func init() {
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	logfile, errs := os.OpenFile("log/rcsfileregistry.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	//log.SetOutput(logfile)
	log.SetOutput(io.MultiWriter(os.Stdout, logfile))
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	FileReg = make(map[string]*FileInfo)

	//处理配置文件
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
upaddr             = 127.0.0.1:8096
downaddr        = 0.0.0.0:8098`
	cf := utils.HandleConfigFile("cfg/rcsfileregistry.ini", defcfg)
	upaddr = cf.MustValue("BASE", "upaddr")
	downaddr = cf.MustValue("BASE", "downaddr")
	buildRegistry(basedir)
	log.Println("build file Registry done,init ok!")

}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	go func() {
		mysm := http.NewServeMux()
		mysm.HandleFunc("/", downld_func)
		log.Println("Start download service:", downaddr)
		log.Fatal(http.ListenAndServe(downaddr, mysm))
	}()
	http.HandleFunc("/upload", upload_func)
	log.Println("Start upload service:", upaddr)
	log.Fatal(http.ListenAndServe(upaddr, nil))

}
func upload_func(w http.ResponseWriter, r *http.Request) {
	rs := new(FileInfo)
	if r.Method == "GET" {
		log.Println("Got request:", r.URL)
		fmd5 := r.URL.Query().Get("fmd5")
		if fmd5 == "" {
			SendResp(rs, w)
			return
		}
		log.Println("fmd5:", fmd5)
		rs = getfilei(fmd5) //rs==nil
		if rs == nil {
			rs = new(FileInfo)
		}
		SendResp(rs, w)
		return
	}
	if r.Method == "POST" {
		r.ParseMultipartForm(10 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			log.Println(err)
			SendResp(rs, w)
			return
		}
		defer file.Close()
		timedir := time.Now().Format("2006-01-02")
		fmd5 := r.URL.Query().Get("fmd5")
		if fmd5 == "" {
			SendResp(rs, w)
			return
		}
		if err := os.MkdirAll(filepath.Join(basedir, timedir, fmd5), 0755); err != nil {
			log.Println(err)
			SendResp(rs, w)
			return
		}
		f, err := os.OpenFile(filepath.Join(basedir, timedir, fmd5, filepath.Base(handler.Filename)), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Println(err)
			SendResp(rs, w)
			return
		}
		defer f.Close()

		md5h := md5.New()
		defer md5h.Reset()
		sz, err1 := io.Copy(io.MultiWriter(f, md5h), file)
		if err1 != nil {
			log.Println(err1)
			//rs.ErrStatus = err1.Error()
			SendResp(rs, w)
			return
		} else {
			rs.Md5str = hex.EncodeToString(md5h.Sum(nil))
			rs.Url = "http://" + downaddr + "/" + basedir + "/" + timedir + "/" + fmd5 + "/" + filepath.Base(f.Name())
			rs.Size = sz
			fileadd(rs.Md5str, rs)
			SendResp(rs, w)
			log.Println("Recv file successfully,create url done:", rs.Url)
			return
		}
	}
}
func downld_func(w http.ResponseWriter, r *http.Request) {
	log.Println("Got a downfile request:", r.URL.String())
	fp := "." + r.RequestURI
	w.Header().Set("Connection", "close")
	//w.Header().Set("Content-Encoding", "gzip") //请求方需要设置:"Accept-Encoding", "gzip"
	http.ServeFile(w, r, fp)
}
func SendResp(rs *FileInfo, w http.ResponseWriter) {
	resp, e := json.Marshal(*rs)
	if e != nil {
		log.Println(e)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	if _, e := w.Write(resp); e != nil {
	}
}
func buildRegistry(dir string) {
	//filelist := make([]string, 0)
	wf := func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		fi := createFileInfo(path)
		//filelist = append(filelist, path)
		fileadd(fi.Md5str, fi)
		return nil
	}
	err := filepath.Walk(dir, wf)
	if err != nil {
		log.Printf("filepath.Walk() returned %v\n", err)
	}
	return
}
func createFileInfo(path string) *FileInfo {
	fi := new(FileInfo)
	fi.Url = "http://" + downaddr + "/" + strings.Replace(path, `\`, `/`, -1)
	sz, md, err := utils.FileSizeAndMd5(path)
	if err != nil {
		return nil
	}
	fi.Md5str = md
	fi.Size = sz
	return fi
}
