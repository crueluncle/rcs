package modules

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"rcs/utils"
	"runtime/debug"
	"strings"
	"time"
)

type Mqconfig struct {
	mqUri,
	exChangeName,
	queueName,
	rKey string
}

func NewMqconfig(mqUri, exChangeName, queueName, rKey string) *Mqconfig {
	return &Mqconfig{
		mqUri:        mqUri,
		exChangeName: exChangeName,
		queueName:    queueName,
		rKey:         rKey,
	}

}

type transfer struct {
	p,
	c1,
	c2 *utils.Pdcser
	cdr utils.Codecer
}

func Newtransfer(p, c1, c2 *utils.Pdcser) *transfer {
	if p == nil || c1 == nil || c2 == nil {
		return nil
	}
	return &transfer{
		p:  p,
		c1: c1,
		c2: c2,
	}
}

func (mm transfer) HandleConn(conn *net.TCPConn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	//log.Println("conn:", conn.LocalAddr(), "-->", conn.RemoteAddr())
	mm.cdr = utils.NewCodecer(conn)
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
func (mm transfer) sendResp(ctx context.Context) {
	//defer mm.cdr.Close()
	/*select {
	case <-ctx.Done():
		return
	default:
		err := mm.cdr.Write(r)
		if err != nil {
			log.Println("Send taskresponse to master failed:", err, r.Runid)
			_ = mm.cdr.Close()
			return
		}
		log.Println("Send taskresponse to master done:", r.Runid)
	}*/
	var msgs = make(chan []byte, 64)
	var (
		taskresp = new(utils.RcsTaskResp)
		err      error
	)
	go func() {
		log.Fatalln(mm.c1.Comsumer(msgs))
	}()
	for taskdata := range msgs {
		select {
		case <-ctx.Done():
			msgs <- taskdata
			return
		default:
			err = json.Unmarshal(taskdata, taskresp)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("Fetch a taskresp msg from mq:", taskresp.Runid)
			err = mm.cdr.Write(taskresp)
			if err != nil {
				log.Println("Send taskresponse to master failed:", err, taskresp.Runid)
				_ = mm.cdr.Close()
				return
			}
			log.Println("Send taskresponse to master done:", taskresp.Runid)

		}
	}

}
func (mm transfer) syncagent(ctx context.Context, jip string) {
	var msgs = make(chan []byte, 64)
	var (
		syncinfo = new(utils.AgentSyncMsg)
		err      error
	)
	go func() {
		log.Fatalln(mm.c2.Comsumer(msgs))
	}()
	for syncdata := range msgs {
		select {
		case <-ctx.Done():
			msgs <- syncdata
			return
		default:
			err = json.Unmarshal(syncdata, syncinfo)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("Fetch a syncinfo msg from mq:", syncinfo.Agentip)
			syncinfo.Jip = jip
			err = mm.cdr.Write(syncinfo)
			if err != nil {
				log.Println("Send syncinfo to master failed:", err, syncinfo.Agentip)
				_ = mm.cdr.Close()
				return
			}
			log.Println("Send syncinfo to master done:", syncinfo.Agentip)
		}
	}

}
func (mm transfer) getTask(ctx context.Context) {
	var tasks = make(chan interface{}, 64)
	var taskdata []byte
	var err error
	go func() {
		if mm.cdr != nil { //读出task放到tasks
			err := mm.cdr.Read(tasks)
			if err != nil {
				log.Println(err)
				_ = mm.cdr.Close()
				return
			}
		}
	}()
	for t := range tasks {
		select {
		case <-ctx.Done():
			tasks <- t
			return
		default:
			if v, ok := t.(*utils.RcsTaskReqJson); ok {
				taskdata, err = json.Marshal(v)
				if err != nil {
					log.Print(err)
					continue
				}
				err = mm.p.Publish(taskdata)
				if err != nil {
					log.Print(err)
					continue
				}
			}
		}
	}

}
