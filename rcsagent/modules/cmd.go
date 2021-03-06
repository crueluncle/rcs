package modules

import (
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (seb Cmd_run_req) Handle(res *Atomicresponse) error {
	//exec command on agent,donot support commands link like '|' '&' '>' etc..
	var outstd, errstd bytes.Buffer
	var resStderr string
	var command = exec.Command(seb.Cmd, seb.CmdArgs...)

	command.Stderr = &errstd //the stderr of the scripts
	command.Stdout = &outstd //the stdout of the scripts
	err := command.Run()
	/*this 'err' just indicate the execution status of the last line of the \
	scripts,not the status during the executing(maybe some errors occur during the executing),so we should judge the 'err' and the 'command.Stderr' content both
	*/
	if err != nil {
		resStderr = err.Error() + errstd.String()
	} else {
		resStderr = errstd.String()
	}
	if resStderr == "" { //truely correctness, means 'command.Run()' is ok,and no error occur about the script itself
		res.Flag = true
		res.Result = outstd.String()
	} else {
		res.Flag = false
		res.Result = resStderr
	}
	return nil
}

//impliments Atomicrequest interface
func (seb Cmd_script_req) Handle(res *Atomicresponse) error {
	/*
		Script  execute scripts from remote
		1.firstly ,download the script file
		2.and then execute the  script file locally
		3.the response content may has chinese charset of gbk,the caller should be translate it to uft-8 before print it,like use 'github.com/qiniu/iconv' module
	*/
	tmpfiledir := os.TempDir()
	u, err := url.Parse(seb.FileUrl)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	if err := Downloadfilefromurl(seb.FileUrl, seb.FileMd5, tmpfiledir); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	uri := u.RequestURI()
	scriptfilepath := filepath.Join(tmpfiledir, filepath.Base(strings.Split(uri, `?`)[0]))
	var command *exec.Cmd
	switch seb.Stype {
	//case "bat":
	//case "shell":
	case `shell=py`:
		args := []string{scriptfilepath}
		command = exec.Command("python", append(args, seb.ScriptArgs...)...)
	case `shell=pws`:
	case `shell=perl`:
	default:
		command = exec.Command(scriptfilepath, seb.ScriptArgs...)
	}

	var outstd, errstd bytes.Buffer
	var resStderr string
	command.Stderr = &errstd //the stderr of the scripts
	command.Stdout = &outstd //the stdout of the scripts
	err = command.Run()
	/*this 'err' just indicate the execution status of the last line of the \
	scripts,not the status during the executing(maybe some errors occur during the executing),so we should judge the 'err' and the 'command.Stderr' content both
	*/
	if err != nil {
		resStderr = err.Error() + errstd.String()
	} else {
		resStderr = errstd.String()
	}
	if resStderr == "" { //truely correctness, means 'command.Run()' is ok,and no error occur about the script itself
		res.Flag = true
		res.Result = outstd.String()
	} else {
		res.Flag = false
		res.Result = resStderr
	}
	return nil
}
