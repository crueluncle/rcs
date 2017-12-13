package modules

type Os struct { //Os模块,支持os相关操作,方法:Restart,Shutdown,Setpwd
	/*inner execution module 'Os',that support some operation of Os level,like:
	Restart()
	Shutdown()
	Setpwd()
	*/
}
type Os_restart_req struct {
	delay       bool
	delaysecond int64
}
type Os_shutdown_req struct {
	delay       bool
	delaysecond int64
}
type Os_setpwd_req struct {
	username string
	passwd   string
}

func (f Os) Restart(seb Os_restart_req, res *Atomicresponse) error {
	if err := restartos(seb.delay, seb.delaysecond); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
func (f Os) Shutdown(seb Os_shutdown_req, res *Atomicresponse) error {
	if err := shutdownos(seb.delay, seb.delaysecond); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
func (f Os) Setpwd(seb Os_setpwd_req, res *Atomicresponse) error {
	if err := setpasswd(seb.username, seb.passwd); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
