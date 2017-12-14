package modules

import (
	"encoding/json"
)

//define all atomic request structs in here
type File_push_req struct { //only support single file
	Sfileurl, Sfilemd5 string
	DstPath            string
}
type File_pull_req struct { //only support single file
	Sfilepath string
	Dstdir    string
}
type File_cp_req struct {
	Sfilepath string //recursive if it`s a directory,
	Dfilepath string //if exists,overwrite
	Wodir     bool   //only used when Sfilepath is a directory,false:copy the whole directory and it`s bellows,true:only copy the directory`s bellows
}
type File_del_req struct {
	Sfilepath string //recursive if it`s a directory
	Wobak     bool   //without backup,false:with backup,true:without backup
}
type File_grep_req struct {
	Sfilepath  string
	Patternstr string //regular expression
}
type File_replace_req struct {
	Sfilepath  string //must be single file,replace the whole file
	Patternstr string //regular expression
	Repltext   string
}
type File_mreplace_req struct {
	Sfiledir           string //specify directory
	Filenamepatternstr string //specify the filename regular expression in the 'Sfiledir' field
	Patternstr         string //regular expression
	Repltext           string
}
type File_md5sum_req struct {
	Sfilepath string //if directory,compute the md5sum of all the files in the directory
}
type File_ckmd5sum_req struct { //similar to md5sum -c md5file
	Md5filepath string
}

//==============================================

type Cmd_script_req struct {
	FileUrl    string
	FileMd5    string
	ScriptArgs []string
}

//==============================================

type Os_restart_req struct {
	delay       bool
	delaysecond int64
}
type Os_shutdown_req struct {
	delay       bool
	delaysecond int64
}
type Os_setpwd_req struct {
	username string
	passwd   string
}

//==============================================
type Firewall_setrules_req struct {
	Rulename []string
	Op       uint8 //at present,just support the code defined in 'const'
}

//==============================================
type Process_stop_req struct {
	Imagename []string
	Doforce   bool
}

//==============================================
const (
	File_push_req_tp = iota
	File_pull_req_tp
	File_cp_req_tp
	File_del_req_tp
	File_grep_req_tp
	File_replace_req_tp
	File_mreplace_req_tp
	File_md5sum_req_tp
	File_ckmd5sum_req_tp
	Cmd_script_req_tp
	Os_restart_req_tp
	Os_shutdown_req_tp
	Os_setpwd_req_tp
	Firewall_setrules_req_tp
	Process_stop_req_tp
)

type Atomicrequest struct { //uint8=[0-255]
	Tp        uint8           //indicate which struct(type of the atomic request) above should  the 'AtomicReq'  be Unmarshaled into
	AtomicReq json.RawMessage //the json-raw-msg of all the  atomic request above
}
type Atomicresponse struct { //atomic response of all atomic request
	Flag   bool
	Result string
}
