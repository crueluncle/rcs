package main

import (
	"log"
	"net"
	"net/rpc"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"time"
)

type TestrpcServer struct {
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	var trs TestrpcServer
	if _, ams := utils.NewTServer("0.0.0.0:9529", trs); ams != nil {
		log.Fatalln(ams.Serve())
	}

}
func (am TestrpcServer) HandleConn(conn *net.TCPConn) error {
	rcli := rpc.NewClient(conn)
	resp := new(modules.Atomicresponse)

	args := new(modules.Firewall_setrules_req)
	args.Rulename = []string{"carey111", "carey222"}
	args.Op = modules.DisableRule
	for {
		err := rcli.Call("Firewall.Setrules", *args, resp)
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
