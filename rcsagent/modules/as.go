package modules

import (
	"errors"
	"net"
	"net/rpc"
)

func InitRPCserver_win(conn *net.TCPConn) error {
	//register services defined in 'modules' for windows platform agent
	defer conn.Close()
	RpcServer := rpc.NewServer()
	err := RpcServer.Register(File{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(Cmd{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(Os{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(Firewall{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(Process{})
	if err != nil {
		return err
	}
	/*
		err = RpcServer.Register(Rcs{})
		if err != nil {
			return err
		}
		err = RpcServer.Register(Archive{})
		if err != nil {
			return err
		}


	*/
	RpcServer.ServeConn(conn)
	return errors.New("RpcServer exit.")
}

func InitRPCserver_unix(conn *net.TCPConn) error {
	//register services defined in 'modules' for unix platform agent
	defer conn.Close()
	RpcServer := rpc.NewServer()
	err := RpcServer.Register(File{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(Cmd{})
	if err != nil {
		return err
	}
	err = RpcServer.Register(Os{})
	if err != nil {
		return err
	}
	/*
		err = RpcServer.Register(Rcs{})
		if err != nil {
			return err
		}
		err = RpcServer.Register(Archive{})
		if err != nil {
			return err
		}

	*/
	RpcServer.ServeConn(conn)
	return errors.New("RpcServer exit.")
}
