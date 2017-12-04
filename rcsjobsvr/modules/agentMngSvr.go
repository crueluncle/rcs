package modules

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"rcs/rcsagent"
	"runtime/debug"
	"sync"
	"time"
)

type agentEntry struct {
	conn  *net.TCPConn
	rpcli *rpc.Client
	doing *sync.Mutex
}
type acontainer map[string]*agentEntry

type agentMngSvr struct {
	ctnlock           *sync.RWMutex //protect 'agentCtn'
	agentCtn          acontainer
	nodeRouteMap      *sync.Map
	keepAliveDuration int
	routeId           uint16
}

func NewAgentMngSvr(ackt int, routeId uint16, rm *sync.Map) *agentMngSvr {
	am := new(agentMngSvr)
	go func() {
		for {
			log.Println("Agents counts:", len(am.agentCtn))
			time.Sleep(time.Minute)
		}
	}()

	am.ctnlock = new(sync.RWMutex)
	am.agentCtn = acontainer(make(map[string]*agentEntry))
	am.nodeRouteMap = rm
	am.keepAliveDuration = ackt
	am.routeId = routeId
	return am
}
func (am agentMngSvr) HandleConn(conn *net.TCPConn) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	//ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
	ip := conn.RemoteAddr().String()
	if ai := am.Getagent(ip); ai != nil {
		if err := conn.Close(); err != nil {
			return err
		}
		return errors.New("Agent regist conflict, closing the connection!")
	}
	rcli := rpc.NewClient(conn)
	ai := agentEntry{conn, rcli, new(sync.Mutex)}
	am.addagent(ip, &ai)

	resp := new(rcsagent.RpcCallResponse)
	args := new(rcsagent.Rcs_HeartBeat_Req)
	var argss rcsagent.RpcCallRequest = args
	var err error
	for {
		err = rcli.Call("ModuleService.Run", &argss, resp)
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
	am.nodeRouteMap.Store(key, am.routeId)
}
func (am *agentMngSvr) delagent(key string) {
	ai := am.Getagent(key)
	am.ctnlock.Lock()

	if ai != nil {
		if e := ai.rpcli.Close(); e != nil {
			//log.Println(e)
		}
		if e := ai.conn.Close(); e != nil {
			//log.Println(e)
		}
		delete(am.agentCtn, key)

	}
	am.ctnlock.Unlock()
	am.nodeRouteMap.Delete(key)
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
