package modules

import (
	"context"
	"log"
	"net"
	"os"
	"rcs/utils"
	"runtime/debug"
	"strings"
	"time"
)

type masterMngSvr struct {
	tasks    chan<- interface{}
	resps    chan *utils.RcsTaskResp
	syncchan chan *utils.AgentSyncMsg
	cdr      utils.Codecer
}

func NewMasterManager(tchan chan<- interface{}, respchan chan *utils.RcsTaskResp, syncch chan *utils.AgentSyncMsg) *masterMngSvr {
	return &masterMngSvr{
		tasks:    tchan,
		resps:    respchan,
		syncchan: syncch,
	}
}

func (mm masterMngSvr) HandleConn(conn *net.TCPConn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	//log.Println("conn:", conn.LocalAddr(), "-->", conn.RemoteAddr())
	decoder := utils.NewCodecer(conn)
	mm.cdr = decoder
	cxt, cancel := context.WithCancel(context.Background())
	jsvip := strings.Split(conn.LocalAddr().String(), ":")[0]
	go mm.sendResp(cxt)
	go mm.syncagent(cxt, jsvip)
	go mm.getTask(cxt)

	km := new(utils.KeepaliveMsg)
	var err error
	for i := 0; ; i++ {
		km.Id = "hello"
		km.Sn = i % 1000
		err = mm.cdr.Write(km)
		if err != nil {
			break
		}
		time.Sleep(time.Second * 3)
	}
	cancel()
	_ = mm.cdr.Close()
	return err
}
func (mm masterMngSvr) sendResp(ctx context.Context) {
	//defer mm.cdr.Close()
	for r := range mm.resps {
		select {
		case <-ctx.Done():
			mm.resps <- r
			return
		default:
			err := mm.cdr.Write(r)
			if err != nil {
				log.Println("Send taskresponse to master failed:", err, r.Runid)
				mm.resps <- r
				_ = mm.cdr.Close()
				return
			}
			log.Println("Send taskresponse to master done:", r.Runid)
		}
	}
}
func (mm masterMngSvr) syncagent(ctx context.Context, jip string) {
	//defer mm.cdr.Close()
	for r := range mm.syncchan {
		select {
		case <-ctx.Done():
			mm.syncchan <- r
			return
		default:
			r.Jip = jip
			err := mm.cdr.Write(r)
			if err != nil {
				log.Println(err)
				mm.syncchan <- r
				_ = mm.cdr.Close()
				return
			}
			log.Println("Sync agentinfo to master done:", r.Agentip)
		}
	}

}
func (mm masterMngSvr) getTask(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if mm.cdr != nil { //读出task放到tasks
				err := mm.cdr.Read(mm.tasks)
				if err != nil {
					log.Println(err)
					_ = mm.cdr.Close()
					return
				}
			} else {
				return
			}
		}
	}

}
