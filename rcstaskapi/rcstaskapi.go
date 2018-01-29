package main

//接收调用方传递过来的task执行态的json串,解析为task对象并存入redis队列,给调用方异步返回json消息
import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"rcs/utils"
	"runtime/debug"

	"github.com/garyburd/redigo/redis"
	"github.com/pborman/uuid"
)

var apiServer_addr string
var logfile *os.File
var redisClient *redis.Pool
var errs error
var (
	redisconstr,
	redispass string
	redisDB,
	rMaxIdle,
	rMaxActive int
)

func init() {
	if err := os.MkdirAll(`log`, 0666); err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(`cfg`, 0666); err != nil {
		log.Fatalln(err)
	}
	logfile, errs = os.OpenFile("log/rcstaskapi.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
apiServer_addr     = 0.0.0.0:9527
redisconstr = 127.0.0.1:6379
redisDB = 0
redispass   = yourPassword
rMaxIdle    = 10
rMaxActive  = 100`
	cf := utils.HandleConfigFile("cfg/rcstaskapi.ini", defcfg)
	apiServer_addr = cf.MustValue("BASE", "apiServer_addr")
	redisconstr = cf.MustValue("BASE", "redisconstr")
	redisDB = cf.MustInt("BASE", "redisDB")
	redispass = cf.MustValue("BASE", "redispass")
	rMaxIdle = cf.MustInt("BASE", "rMaxIdle")
	rMaxActive = cf.MustInt("BASE", "rMaxActive")
	redisClient, errs = utils.Newredisclient(redisconstr, redispass, redisDB, rMaxIdle, rMaxActive) //for write redis
	if errs != nil {
		log.Fatalln(errs)
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	http.HandleFunc("/runtask", runtask)

	log.Println("Api Server start ok:", apiServer_addr)
	log.Fatal(http.ListenAndServe(apiServer_addr, nil))
}
func runtask(w http.ResponseWriter, r *http.Request) {
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
		if task.Runid != "" { //调用者传过来的必须是"",然后master生存唯一的runid回应给调用者
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
		//	taskreq, err := task.Parse()
		_, err := task.Parse()
		if err != nil {
			log.Println(err)
			rs.ErrStatus = err.Error()
			rs.EncodeJson(w)
		}
		err = utils.WriteTaskinfo(task, redisClient.Get())
		if err != nil {
			log.Println(err)
			rs.ErrStatus = err.Error()
			rs.EncodeJson(w)
		}
		rs.Uuid = runid
		rs.EncodeJson(w)
	} else {
		log.Println("invalid request method!\n")
		rs.ErrStatus = "invalid request method!"
		rs.EncodeJson(w)
	}
}
