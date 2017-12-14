package modules

// for windows platform only
const (
	DisableRule = iota
	EnableRule
	DeleteRule
)

type Firewall struct {
	/*inner module 'Firewall',for operate windows firewall,function:
	Setrules()
	*/
}

func (fw Firewall) Setrules(seb Firewall_setrules_req, res *Atomicresponse) error {
	//Sets new values for properties of a existing rule. ust support windows platform
	if err := setrules(seb.Rulename, seb.Op); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
