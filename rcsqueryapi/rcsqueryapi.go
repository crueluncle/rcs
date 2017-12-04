package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"rcs/utils"
	"runtime/debug"

	"github.com/garyburd/redigo/redis"
)

var (
	apiServer_addr,
	redisconstr,
	redispass string
	redisDB,
	rMaxIdle,
	rMaxActive int
)

var logfile *os.File
var RedisClient *redis.Pool

func init() {
	var errs error
	logfile, errs = os.OpenFile("log/rcsqueryapi.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0777)
	if errs != nil {
		log.Fatal(errs)
	}
	//log.SetOutput(logfile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	log.Println("Version:", utils.Version, " BuildTime:", utils.BuildTime, " Author:", utils.Author)
	defcfg := `;section Base defines some params,'SectionName' in []  must be uniq globally.
[BASE]
apiServer_addr = 0.0.0.0:9528
redisconstr = 127.0.0.1:6379
redisDB     = 0
redispass   = yourPassword
rMaxIdle    = 100
rMaxActive  = 20000`

	cf := utils.HandleConfigFile("cfg/rcsqueryapi.ini", defcfg)
	apiServer_addr = cf.MustValue("BASE", "apiServer_addr")
	redisconstr = cf.MustValue("BASE", "redisconstr")
	redisDB = cf.MustInt("BASE", "redisDB")
	redispass = cf.MustValue("BASE", "redispass")
	rMaxIdle = cf.MustInt("BASE", "rMaxIdle")
	rMaxActive = cf.MustInt("BASE", "rMaxActive")
	RedisClient, errs = utils.Newredisclient(redisconstr, redispass, redisDB, rMaxIdle, rMaxActive)
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
	http.HandleFunc("/gettasksfnums", getsfnumsfromredis)
	http.HandleFunc("/getagentresult", getagentresultfromredis)
	http.HandleFunc("/getagentresultinsucc", getagentresultinsucc)
	http.HandleFunc("/getagentresultinfail", getagentresultinfail)
	http.HandleFunc("/gettaskresult", getresultfromredis)

	log.Println("Start rekapi...")
	log.Println("RcshttpAPI:query ApiServer start...:", apiServer_addr)
	log.Fatal(http.ListenAndServe(apiServer_addr, nil))
}

func getsfnumsfromredis(w http.ResponseWriter, r *http.Request) {

	log.Println("Got request:", r.URL)
	uuid := r.URL.Query().Get("uuid")
	if len(uuid) != 36 {
	}
	rs := utils.GetSFnumsFromRedis(uuid, RedisClient.Get())
	resp, e := json.Marshal(rs)
	if e != nil {
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	if _, e := w.Write(resp); e != nil {
	}
}
func getagentresultfromredis(w http.ResponseWriter, r *http.Request) {

	log.Println("Got request:", r.URL)
	uuid := r.URL.Query().Get("uuid")
	ip := r.URL.Query().Get("ip")
	rs := utils.GetAgentResultFromRedis(uuid, ip, RedisClient.Get())
	resp, e := json.Marshal(rs)
	if e != nil {
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	if _, e := w.Write(resp); e != nil {
	}
}
func getagentresultinsucc(w http.ResponseWriter, r *http.Request) {

	log.Println("Got request:", r.URL)
	uuid := r.URL.Query().Get("uuid")
	ip := r.URL.Query().Get("ip")
	rs := utils.GetAgentResultInSucc(uuid, ip, RedisClient.Get())
	resp, e := json.Marshal(rs)
	if e != nil {
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	if _, e := w.Write(resp); e != nil {
	}
}
func getagentresultinfail(w http.ResponseWriter, r *http.Request) {

	log.Println("Got request:", r.URL)
	uuid := r.URL.Query().Get("uuid")
	ip := r.URL.Query().Get("ip")
	rs := utils.GetAgentResultInFail(uuid, ip, RedisClient.Get())
	resp, e := json.Marshal(rs)
	if e != nil {
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	if _, e := w.Write(resp); e != nil {
	}
}
func getresultfromredis(w http.ResponseWriter, r *http.Request) {

	log.Println("Got request:", r.URL)
	uuid := r.URL.Query().Get("uuid")
	rs := utils.GetResultFromRedis(uuid, RedisClient.Get())
	resp, e := json.Marshal(rs)
	if e != nil {
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Connection", "close")
	w.Header().Set("Content-Encoding", "gzip")
	if _, e := w.Write(resp); e != nil {
	}
}
