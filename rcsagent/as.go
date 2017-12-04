package rcsagent

import (
	"bytes"
	"errors"
	"log"
	"net"
	"net/rpc"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	//	"time"
)

var tmpfiledir string = "scriptstmpfiledir"

func (seb *Script_Run_Req) Handle(res *RpcCallResponse) error {
	log.Println("handle 1 request:RpcCallRequest ", seb)
	//start := time.Now()
	if err := Downloadfilefromurl(seb.FileUrl, seb.FileMd5, tmpfiledir); err != nil {
		log.Println("downloadfilefromregistry:", err)
		return err
	}
	//log.Println("time spend1:", time.Since(start).Nanoseconds()/1000000)
	u, e := url.Parse(seb.FileUrl)
	if e != nil {
		log.Println(e)
		return e
	}
	uri := u.RequestURI()
	scriptfilepath := filepath.Join(tmpfiledir, filepath.Base(strings.Split(uri, `?`)[0]))
	command := exec.Command(scriptfilepath, seb.ScriptArgs...)
	var outstd, errstd bytes.Buffer
	var resStderr string
	command.Stderr = &errstd
	command.Stdout = &outstd
	//	log.Println("command:", command)
	err := command.Run()
	if err != nil {
		resStderr = err.Error() + errstd.String()
		log.Println("resStderr:", resStderr)
	} else {
		resStderr = errstd.String()
	}
	if resStderr == "" {
		res.Flag = true
		res.Result = outstd.String()
	} else {
		res.Flag = false
		res.Result = resStderr
	}
	//log.Println("time spend:", time.Since(start).Nanoseconds()/1000000)
	return nil
}
func (seb *File_Push_Req) Handle(res *RpcCallResponse) error {
	log.Println("handle 1 request:RpcCallRequest ", seb)
	if err := Downloadfilefromurl(seb.FileUrl, seb.FileMd5, seb.DstPath); err != nil {
		log.Println("downloadfilefromjobsvr:", err)
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = seb.FileMd5
	return nil
}

func (seb *Rcs_Restart_Req) Handle(res *RpcCallResponse) error {

	return nil
}
func (seb *Rcs_Stop_Req) Handle(res *RpcCallResponse) error {

	return nil
}
func (seb *Rcs_Upgrade_Req) Handle(res *RpcCallResponse) error {

	return nil
}
func (seb *Rcs_HeartBeat_Req) Handle(res *RpcCallResponse) error {
	res.Flag = true
	res.Result = seb.Msg
	return nil
}
func (seb *Script_Run_Req) GetFileUrl() string {
	return seb.FileUrl
}
func (seb *File_Push_Req) GetFileUrl() string {
	return seb.FileUrl
}
func (seb *Rcs_Restart_Req) GetFileUrl() string {
	return ""
}
func (seb *Rcs_Stop_Req) GetFileUrl() string {
	return ""
}
func (seb *Rcs_Upgrade_Req) GetFileUrl() string {
	return ""
}
func (seb *Rcs_HeartBeat_Req) GetFileUrl() string {
	return ""
}
func (seb *Script_Run_Req) GetFileMd5() string {
	return seb.FileMd5
}
func (seb *File_Push_Req) GetFileMd5() string {
	return seb.FileMd5
}
func (seb *Rcs_Restart_Req) GetFileMd5() string {
	return ""
}
func (seb *Rcs_Stop_Req) GetFileMd5() string {
	return ""
}
func (seb *Rcs_Upgrade_Req) GetFileMd5() string {
	return ""
}
func (seb *Rcs_HeartBeat_Req) GetFileMd5() string {
	return ""
}
func (seb *Script_Run_Req) SetFileUrl(newurl string) {
	seb.FileUrl = newurl
}
func (seb *File_Push_Req) SetFileUrl(newurl string) {
	seb.FileUrl = newurl
}
func (seb *Rcs_Restart_Req) SetFileUrl(newurl string) {

}
func (seb *Rcs_Stop_Req) SetFileUrl(newurl string) {

}
func (seb *Rcs_Upgrade_Req) SetFileUrl(newurl string) {

}
func (seb *Rcs_HeartBeat_Req) SetFileUrl(newurl string) {

}

type ModuleService struct {
}

func (s ModuleService) Run(seb RpcCallRequest, res *RpcCallResponse) error {
	return seb.Handle(res)
}

func settmpdir() error {
	file, e := exec.LookPath(os.Args[0])
	if e != nil {
		return e
	}
	path, e := filepath.Abs(file)
	if e != nil {
		return e
	}
	tmpfiledir = filepath.Join(filepath.Dir(path), tmpfiledir)
	return nil
}

func StartRPCserver(conn *net.TCPConn) error {
	defer conn.Close()
	//log.Println("tmpfiledir:", tmpfiledir)
	RpcServer := rpc.NewServer()
	err := RpcServer.Register(ModuleService{})
	if err != nil {
		return err
	}
	RpcServer.ServeConn(conn)
	return errors.New("RpcServer exit")
}
func init() {
	e := settmpdir()
	if e != nil {
		log.Fatalln(e)
	}
}
