package modules

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"rcs/utils"
	"runtime/debug"

	"github.com/pborman/uuid"
)

type masterapi struct {
	listenaddr string
	tasklist   chan<- *utils.RcsTaskReq
}

func NewMasterapi(addr string, tl chan<- *utils.RcsTaskReq) *masterapi {
	return &masterapi{
		listenaddr: addr,
		tasklist:   tl,
	}
}
func (ma *masterapi) Serve() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	http.HandleFunc("/runtask", ma.runtask)

	log.Println("Api Server start ok:", ma.listenaddr)
	log.Fatal(http.ListenAndServe(ma.listenaddr, nil))
}
func (ma *masterapi) runtask(w http.ResponseWriter, r *http.Request) {
	/*访问示例
	curl -X POST -d "{\"Runid\": 0,\"Targets\": [\"127.0.0.1\"],\"Req\": {\"ScriptFileUrl\": \"http://115.182.81.164/pub/test.bat\",\"ScriptFileMd5\": \"664d0430ee33458602e580520841a2d4\",\"ScriptArgs\": [\"-a\",\"-b\"]}}"  http://127.0.0.1:9999/runscript
		  	success:0
		  	failed:some string
	*/
	//log.Println("PORFORM:GGGGOT a call from apicaller!!")
	jsondec := json.NewDecoder(r.Body)
	rs := new(utils.MasterApiResp)
	task := new(utils.RcsTaskReqJson)
	if r.Method == "POST" {
		if e := jsondec.Decode(task); e != nil {
			log.Println(e)
			rs.ErrStatus = e.Error()
			rs.EncodeJson(w)
		}
		if e := r.Body.Close(); e != nil {
			log.Println(e)
		}
		if task.Runid != "" {
			log.Println("original runid is invalid!")
			rs.ErrStatus = "original runid is invalid!"
			rs.EncodeJson(w)
		}
		runid := uuid.NewUUID().String()
		if runid == "" {
			log.Println("uuid.NewUUID():get runid failed")
			rs.ErrStatus = "uuid.NewUUID():get runid failed"
			rs.EncodeJson(w)
		}
		task.Runid = runid
		ma.tasklist <- task.Parse()
		rs.Uuid = runid
		rs.EncodeJson(w)
	} else {
		log.Println("invalid request method!\n")
		rs.ErrStatus = "invalid request method!"
		rs.EncodeJson(w)
	}
}
