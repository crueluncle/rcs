package modules

type Os struct { //Os模块,支持os相关操作,方法:Restart,Shutdown,Setpwd
	/*inner execution module 'Os',that support some operation of Os level,like:
	Restart()
	Shutdown()
	Setpwd()
	*/
}

func (seb Os_restart_req) Handle(res *Atomicresponse) error {
	if err := restartos(seb.Delay, seb.Delaysecond); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
func (seb Os_shutdown_req) Handle(res *Atomicresponse) error {
	if err := shutdownos(seb.Delay, seb.Delaysecond); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
func (seb Os_setpwd_req) Handle(res *Atomicresponse) error {
	if err := setpasswd(seb.Username, seb.Passwd); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
