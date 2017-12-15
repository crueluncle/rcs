package modules

import (
	"errors"
	"net"
	"net/rpc"
)

type Service struct {
	//rcp service for out
}

func (s Service) Call(req Atomicrequest, res *Atomicresponse) error {
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
