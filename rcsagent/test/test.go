package main

import (
	"fmt"
	"log"
	"rcs/rcsagent/modules"
)

func main() {

	log.SetFlags(log.Llongfile)

	req := new(modules.Cmd_script_req)
	req.FileUrl = `http://120.92.94.165/pub/upload/test.bat`
	req.FileMd5 = `fa0399a99d00b33eb096627d8e5d6e6b`
	resp := new(modules.Atomicresponse)
	f := new(modules.Cmd)
	if e := f.Script(*req, resp); e != nil {
		log.Fatalln(e)
	}
	log.Println(resp.Flag)
	fmt.Println(resp.Result)

}
