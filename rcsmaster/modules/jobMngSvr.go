package modules

import (
	"errors"
	"log"
	"net"
	"os"
	"rcs/utils"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
)

type jobsvrEntry struct {
	conn     *net.TCPConn
	connrwer utils.Codecer
}
type jContainer map[string]*jobsvrEntry
type jobsvrManager struct {
	ctnlocker *sync.RWMutex //protect jobsvrCtn
	jobsvrCtn jContainer
	msgchan   chan interface{}
	tasklist  <-chan *utils.RcsTaskReq
	f         func() redis.Conn
}

func NewJobsvrManager(getredisconfunc func() redis.Conn, tasklistchan <-chan *utils.RcsTaskReq) *jobsvrManager {
	jsm := new(jobsvrManager)
	go func() {
		for {
			log.Println("jobsvr counts:", len(jsm.jobsvrCtn))
			time.Sleep(time.Minute)
		}
	}()

	jsm.ctnlocker = new(sync.RWMutex)
	jsm.jobsvrCtn = jContainer(make(map[string]*jobsvrEntry))
	jsm.msgchan = make(chan interface{}, 32)
	jsm.tasklist = tasklistchan
	jsm.f = getredisconfunc
	return jsm
}
func (jsm jobsvrManager) HandleConn(conn *net.TCPConn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	connrwer := utils.NewCodecer(conn)
	defer connrwer.Close()

	ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
	if _, ok := jsm.jobsvrCtn[ip]; ok { //不允许jobsvr ip重复，包括机器本身ip冲突或者1台机器上起多个jobsvr进程
		if err := conn.Close(); err != nil {
			return err
		}
		return errors.New("Jobsvr regist conflict,closing the connection!")
	}
	jsm.addjobsvr(ip, &jobsvrEntry{conn, connrwer})
	go jsm.broadcastTask()
	go jsm.saveResponse()
	err := connrwer.Read(jsm.msgchan)
	jsm.deljobsvr(ip)
	log.Println("Close the connection from jobsvr:", err, conn.LocalAddr(), "<----->", conn.RemoteAddr())
	return err
}

func (jsm *jobsvrManager) addjobsvr(key string, val *jobsvrEntry) {
	jsm.ctnlocker.Lock()
	defer jsm.ctnlocker.Unlock()
	jsm.jobsvrCtn[key] = val
}
func (jsm *jobsvrManager) deljobsvr(key string) {
	ai := jsm.getjobsvr(key)
	jsm.ctnlocker.Lock()
	defer jsm.ctnlocker.Unlock()
	if ai != nil {
		if e := ai.conn.Close(); e != nil {
		}
		delete(jsm.jobsvrCtn, key)
	}
}
func (jsm *jobsvrManager) getjobsvr(key string) *jobsvrEntry {
	jsm.ctnlocker.RLock()
	defer jsm.ctnlocker.RUnlock()
	r, ok := jsm.jobsvrCtn[key]
	if ok == false {
		return nil
	}
	return r
}
func (jsm *jobsvrManager) broadcastTask() error {
	if len(jsm.jobsvrCtn) == 0 {
		return errors.New("No jobsvr exist!")
	}
	for task := range jsm.tasklist {
		for ip, jsinfo := range jsm.jobsvrCtn {
			if jsinfo == nil {
				return errors.New("Have invalid jobsvr!")
			}
			err := jsinfo.connrwer.Write(task)
			if err != nil {
				jsm.deljobsvr(ip)
				log.Println("Send task to jobsvr error:", err)
				return err
			}
			log.Println("Send one task to jobsvr done!")
		}
	}
	return nil
}
func (jsm *jobsvrManager) saveResponse() {

	for msg := range jsm.msgchan {
		if _, ok := msg.(*utils.KeepaliveMsg); ok { //结接收到心跳消息,什么也不干
		} else if res, ok := msg.(*utils.RcsTaskResp); ok { //接收到响应消息
			var i int
			for i = 0; i < 3; i++ {
				e := utils.Writeresponserun(res, jsm.f())
				if e == nil {
					break
				} else { //如果写失败,重试3次
					//log.Println("%s,%s,%d response Write2redis failed,continue:", e, res.Runid)
					time.Sleep(time.Second)
					continue
				}
			}
			if i == 3 {
				log.Println("One response Write2redis failed:", res.Runid, res.AgentIP)
				//ch <- res //失败3次的结果重新放入队列
				continue //
			}
			log.Println("Got one response msg from jobsvr and put in db success!", res.Runid, res.AgentIP)
		}
	}

}
