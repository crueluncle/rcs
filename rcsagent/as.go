package rcsagent

import (
	"errors"
	"net"
	"net/rpc"
	"rcs/rcsagent/modules"
)

func InitRPCserver(conn *net.TCPConn) error {
	//register services defined in 'modules'
	defer conn.Close()
	RpcServer := rpc.NewServer()
	err := RpcServer.Register(modules.File{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(modules.Cmd{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(modules.Os{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(modules.Rcs{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(modules.Archive{})
	if err != nil {
		return err
	}
	RpcServer.ServeConn(conn)
	return errors.New("RpcServer exit.")
}
