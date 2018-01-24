package modules

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"reflect"
)

type Service struct {
	//rcp service for out
}

func (s Service) Call(req Atomicrequest, res *Atomicresponse) error {
	log.Println("Got Atomicrequest call:", reflect.TypeOf(req).String(), req)
	return req.Handle(res)
}
func InitRPCserver(conn *net.TCPConn) error {
	//register services
	defer conn.Close()
	RpcServer := rpc.NewServer()
	err := RpcServer.Register(Service{})
	if err != nil {
		return err
	}
	RpcServer.ServeConn(conn)
	return errors.New("RpcServer exit.")
}
