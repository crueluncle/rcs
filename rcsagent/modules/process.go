package modules

// for windows platform only

type Process struct {
	/*inner module 'Archive',for file or directory archive,function:
	 */
}
type Process_stop_req struct {
	Imagename []string
	Doforce   bool
}

func (p Process) Stop(seb Process_stop_req, res *Atomicresponse) error {
	if err := stopprocess(seb.Imagename, seb.Doforce); err != nil {
		res.Flag = false
		res.Result = err.Error()
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
