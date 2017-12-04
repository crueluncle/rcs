//define the tcpServer and tcpClient framwork...//
/* bellow shows how to make a tcpServer use TServer framwork,and the tcpClient is similar
type mysvr struct { //定义业务服务器
	A     string
	B     int
	msgch chan interface{}
}

func (ms mysvr) HandleConn(conn *net.TCPConn) error {//定义业务处理器,满足THandler接口：任意业务处理逻辑写在这个方法中
	codecer := rek.NewCodecer(conn)
	defer codecer.Close()
	go func() error { //handle msg and send msg
		for v := range ms.msgch {
			log.Println("Got a msg from client:", v, "echo msg")
			if e := codecer.Write(v); e != nil {
				log.Println(e)
				return e
			}
		}
		return nil
	}()
	return codecer.Read(ms.msgch) //read msg from conn
}

func main() {
	ms := mysvr{msgch: make(chan interface{}, 32)}
	if _, ts := rek.NewTServer("127.0.0.1:22222", ms); ts != nil {
		log.Fatalln(ts.Serve())
	}

}

*/
package utils

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"time"
)

type THandler interface {
	HandleConn(conn *net.TCPConn) error
}
type TFunc func(conn *net.TCPConn) error

func (bf TFunc) HandleConn(conn *net.TCPConn) error {
	return bf(conn)
}

type TServer struct {
	listenAddr string
	THandler
	//keepAlive bool
	//ctx       context.Context
}

func NewTServer(listenAddr string, hf THandler) (error, *TServer) {
	ts := new(TServer)
	if _, err := net.ResolveTCPAddr("tcp", listenAddr); err != nil {
		return err, nil
	}
	if hf == nil {
		return errors.New("handfunc is nil"), nil
	}
	ts.listenAddr = listenAddr
	ts.THandler = hf
	//	ts.keepAlive = ka
	//ts.ctx = ctx
	return nil, ts
}
func (ts TServer) Serve() error {
	addr, _ := net.ResolveTCPAddr("tcp", ts.listenAddr)
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	log.Println("Server Listening:", ts.listenAddr)
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Println("TServer.Serve:", err)
			continue
		}
		log.Println("Server accept a connection :", conn.RemoteAddr().String())
		go func(conn *net.TCPConn) {
			e := ts.HandleConn(conn)
			log.Println("Server end a connection :", conn.RemoteAddr().String(), e)
		}(conn)
	}
	return nil
}

type TClient struct {
	connectAddr            string
	reconnectDuration      int  //重连时间间隔,单位秒
	reconnectTimes         int  //重连次数,0为无限重连
	reconnectAfterTerminal bool //连接完毕或对端关闭后，是否不退出程序继续重连
	THandler
}

func NewTClient(cna string, rcd, rct int, reAT bool, hf THandler) (error, *TClient) {
	tc := new(TClient)
	if _, err := net.ResolveTCPAddr("tcp", cna); err != nil {
		return err, nil
	}
	if hf == nil {
		return errors.New("handfunc is nil"), nil
	}
	tc.connectAddr = cna
	tc.reconnectDuration = rcd
	tc.reconnectTimes = rct
	tc.reconnectAfterTerminal = reAT
	tc.THandler = hf
	return nil, tc
}
func (tc TClient) Connect() error {
	addr, _ := net.ResolveTCPAddr("tcp", tc.connectAddr)
	conn := new(net.TCPConn)
	var err error
recon:
	switch tc.reconnectTimes {
	case 0:
		for {
			conn, err = net.DialTCP("tcp", nil, addr)
			if err != nil {
				log.Println("Client Dail error,reconnect..:", err)
				time.Sleep(time.Second * time.Duration(1+rand.New(rand.NewSource(time.Now().UnixNano())).Intn(tc.reconnectDuration)))
				continue
			}
			break
		}
	default:
		var i int
		for i = 0; i < tc.reconnectTimes; i++ {
			conn, err = net.DialTCP("tcp", nil, addr)
			if err != nil {
				log.Println("Client Dail error,reconnect..:", err)
				time.Sleep(time.Second * time.Duration(1+rand.New(rand.NewSource(time.Now().UnixNano())).Intn(tc.reconnectDuration)))
				continue
			}
			break
		}
		if i == tc.reconnectTimes {
			return errors.New("Client connect timeout")
		}
	}
	log.Println("Client connect ok:", conn.RemoteAddr())
	e := tc.HandleConn(conn)
	log.Println("Client connection terminated:", conn.RemoteAddr(), e)
	if tc.reconnectAfterTerminal {
		time.Sleep(time.Second * time.Duration(tc.reconnectDuration))
		goto recon
	}
	return e
}
