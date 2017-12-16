package main

import (
	"log"
	"net"
	"net/rpc"

	"encoding/gob"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"time"
)

type TestrpcServer struct {
}

func init() {
	gob.Register(&modules.File_push_req{})
	gob.Register(&modules.File_pull_req{})
	gob.Register(&modules.File_cp_req{})
	gob.Register(&modules.File_del_req{})
	gob.Register(&modules.File_grep_req{})
	gob.Register(&modules.File_replace_req{})
	gob.Register(&modules.File_mreplace_req{})
	gob.Register(&modules.File_md5sum_req{})
	gob.Register(&modules.File_ckmd5sum_req{})
	gob.Register(&modules.Cmd_script_req{})
	gob.Register(&modules.Os_restart_req{})
	gob.Register(&modules.Os_shutdown_req{})
	gob.Register(&modules.Os_setpwd_req{})
	gob.Register(&modules.Firewall_set_req{})
	gob.Register(&modules.Process_stop_req{})
	gob.Register(&modules.Rcs_ping_req{})
}
func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	var trs TestrpcServer
	if _, ams := utils.NewTServer("0.0.0.0:9529", trs); ams != nil {
		log.Fatalln(ams.Serve())
	}

}

//D:\\PGP\\games D:\\PGP\\bak\ false
func (am TestrpcServer) HandleConn(conn *net.TCPConn) error {
	rcli := rpc.NewClient(conn)
	resp := new(modules.Atomicresponse)

	args := new(modules.File_cp_req)
	args.Sfilepath = `D:\Install`
	args.Dfilepath = `D:\InstallBak`
	var argss modules.Atomicrequest
	argss = args
	for {
		err := rcli.Call("Service.Call", &argss, resp)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(resp.Flag)
		log.Println(resp.Result)
		time.Sleep(time.Minute)
	}
	return nil
}
