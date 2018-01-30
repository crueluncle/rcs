package modules

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"rcs/rcsagent/modules"
	"rcs/utils"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type agentEntry struct {
	conn  *net.TCPConn
	rpcli *rpc.Client
	//doing *sync.Mutex
}
type acontainer map[string]*agentEntry

type agentMngSvr struct {
	ctnlock           *sync.RWMutex //protect 'agentCtn'
	agentCtn          acontainer
	keepAliveDuration int
	syncchan          chan<- *utils.AgentSyncMsg
}

func NewAgentMngSvr(ackt int, ch chan<- *utils.AgentSyncMsg) *agentMngSvr {
	am := new(agentMngSvr)
	go func() {
		for {
			log.Println("Agents counts:", len(am.agentCtn))
			time.Sleep(time.Minute)
		}
	}()
	am.ctnlock = new(sync.RWMutex)
	am.agentCtn = acontainer(make(map[string]*agentEntry))
	am.keepAliveDuration = ackt
	am.syncchan = ch
	return am
}
func (am agentMngSvr) HandleConn(conn *net.TCPConn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
	//ip := conn.RemoteAddr().String() //方便测试并发性能，暂时改为注册格式 ip:port
	if ai := am.Getagent(ip); ai != nil { //new connection replace the old one
		//if err := conn.Close(); err != nil {
		//return err
		//}
		log.Println("Agent reconnecting.")
		am.delagent(ip)
		//	return errors.New("Agent regist conflict, closing the connection!")
	}
	rcli := rpc.NewClient(conn)

	//ai := agentEntry{conn, rcli, new(sync.Mutex)}
	ai := agentEntry{conn, rcli}
	am.addagent(ip, &ai)

	resp := new(modules.Atomicresponse)
	args := new(modules.Rcs_ping_req)
	var argss modules.Atomicrequest = args
	var err error
	for {
		//err = rcli.Call("Service.Call", &argss, resp) //gob对interface类型编解码时,encode和decode需传interface的指针进去
		//if err != nil {
		//break
		//}

		divcall := rcli.Go("Service.Call", &argss, resp, nil) //异步调用并设置超时时间
		select {
		case replaycall := <-divcall.Done:
			if replaycall.Error != nil {
				//resp.Result = resp.Result + replaycall.Error.Error()
				err = replaycall.Error
			}
		case <-time.After(time.Second * time.Duration(am.keepAliveDuration)):
			err = errors.New("rcs.ping:timeout:")
		}
		if err != nil {
			break
		}
		time.Sleep(time.Second * time.Duration(1+rand.New(rand.NewSource(time.Now().UnixNano())).Intn(am.keepAliveDuration)))
	}
	am.delagent(ip)
	return err
}
func (am *agentMngSvr) addagent(key string, val *agentEntry) {
	am.ctnlock.Lock()
	am.agentCtn[key] = val

	am.ctnlock.Unlock()
	am.syncchan <- &utils.AgentSyncMsg{"", "add", key, runtime.GOOS} //sync to master
}
func (am *agentMngSvr) delagent(key string) {
	ai := am.Getagent(key)
	am.syncchan <- &utils.AgentSyncMsg{"", "del", key, ""} //sync to master
	am.ctnlock.Lock()
	if ai != nil {
		delete(am.agentCtn, key)
		if e := ai.rpcli.Close(); e != nil {
			//log.Println(e)
		}
		if e := ai.conn.Close(); e != nil {
			//log.Println(e)
		}
	}
	am.ctnlock.Unlock()

}
func (am *agentMngSvr) Getagent(key string) *agentEntry {
	am.ctnlock.RLock()
	defer am.ctnlock.RUnlock()
	r, ok := am.agentCtn[key]
	if ok == false {
		return nil
	}
	return r
}
