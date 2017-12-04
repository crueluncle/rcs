package modules

import (
	"log"
	"net/http"
	"os"
	"runtime/debug"
)

type fileServer struct {
	listenaddr string
	exportdir  string
}

func NewFileSvr(addr, dir string) *fileServer {
	return &fileServer{
		listenaddr: addr,
		exportdir:  dir,
	}
}
func (fs *fileServer) ServeFile() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Panic info is: ", err, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	if e := os.MkdirAll(fs.exportdir, os.ModeDir); e != nil {
		log.Fatalln(e)
	}
	log.Println("Start server ok:listen:" + fs.listenaddr)

	http.HandleFunc("/", downld_func)
	log.Println("Start filecache service:", fs.listenaddr)
	err := http.ListenAndServe(fs.listenaddr, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
func downld_func(w http.ResponseWriter, r *http.Request) {
	fp := "." + r.RequestURI
	http.ServeFile(w, r, fp)
}
