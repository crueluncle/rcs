package main

import (
	"fmt"
	"log"
	"rcs/rcsagent/modules"
)

func main() {

	log.SetFlags(log.Llongfile)

	req := new(modules.File_mreplace_req)
	req.Sfiledir = `D:\carey`
	req.Filenamepatternstr = "1"
	req.Patternstr = "weiny"
	req.Repltext = "carey"

	resp := new(modules.Atomicresponse)
	f := new(modules.File)
	if e := f.Mreplace(*req, resp); e != nil {
		log.Fatalln(e)
	}
	log.Println(resp.Flag)
	fmt.Println(resp.Result)
}
