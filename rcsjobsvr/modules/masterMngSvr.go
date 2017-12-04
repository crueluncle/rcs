package modules

import (
	"log"
	"os"
	"runtime/debug"
	//	"sync"
	"net"
	"rcs/utils"
	"time"
)

type masterMngSvr struct {
	tasks chan<- interface{}
	resps <-chan *utils.RcsTaskResp
	cdr   utils.Codecer
}

func NewMasterManager(tchan chan<- interface{}, respchan <-chan *utils.RcsTaskResp) *masterMngSvr {
	return &masterMngSvr{
		tasks: tchan,
		resps: respchan,
	}
}

func (mm masterMngSvr) HandleConn(conn *net.TCPConn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	decoder := utils.NewCodecer(conn)
	mm.cdr = decoder
	go mm.keepalive()
	go mm.sendResp()
	if decoder != nil { //读出task放到tasks
		err := decoder.Read(mm.tasks)
		if err != nil {
			_ = decoder.Close()
			return err
		}
	}
	return nil
}
func (mm masterMngSvr) keepalive() {
	km := new(utils.KeepaliveMsg)
	var err error
	for i := 0; ; i++ {
		km.Id = "hello"
		km.Sn = i % 1000
		err = mm.cdr.Write(km)
		if err != nil {
			break
		}
		time.Sleep(time.Second * 5)
	}
	_ = mm.cdr.Close()
}
func (mm masterMngSvr) sendResp() {
	for r := range mm.resps {
		//log.Println("Got a task response:", r)
		err := mm.cdr.Write(r)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("Send taskresponse to master done:", r.Runid)
	}
	_ = mm.cdr.Close()
}
