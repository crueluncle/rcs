package main

import (
	"fmt"
	"log"
	"rcs/rcsagent/modules"
	"strings"

	"github.com/qiniu/iconv"
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
	//log.Println(resp.Flag)
	//fmt.Println(resp.Result)
	respt := strings.TrimSpace(resp.Result)

	cd, err := iconv.Open("utf-8", "gbk") // convert utf-8 to gbk
	if err != nil {
		fmt.Println("iconv.Open failed!")
		return
	}
	defer cd.Close()

	gbk := cd.ConvString(respt)

	fmt.Println(gbk)
}
