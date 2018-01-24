package modules

func (seb Rcs_ping_req) Handle(res *Atomicresponse) error {
	res.Flag = true
	res.Result = "OK"
	return nil
}
