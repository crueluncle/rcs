package main

import (
	"fmt"
	"log"
	"rcs/rcsagent/modules"
	//	"time"
)

func main() {

	log.SetFlags(log.Llongfile)

	req := new(modules.File_del_req)
	req.Sfilepath = `d:\carey111.txt`
	req.Wobak = true
	resp := new(modules.Atomicresponse)
	f := new(modules.File)
	if e := f.Del(*req, resp); e != nil {
		log.Fatalln(e)
	}
	log.Println(resp.Flag)
	fmt.Println(resp.Result)
}
