package modules

/*
	"file.push"   -- File_push_req
	"file.pull"   -- File_pull_req
	"file.cp"     -- File_cp_req
	"file.del"    -- File_del_req
	"file.rename"    -- File_rename_req
	"file.grep"   -- File_grep_req
	"file.replace" --File_replace_req
	"file.mreplace" --File_mreplace_req
	"file.md5sum"   --File_md5sum_req
	"file.ckmd5sum" --File_ckmd5sum_req
	"file.zip"      --File_zip_req
	"file.unzip"	--File_unzip_req
	"cmd.run"       --Cmd_run_req
	"cmd.script"    --Cmd_script_req
	"os.restart"    --Os_restart_req
	"os.shutdown"   -- Os_shutdown_req
	"os.setpwd"     --Os_setpwd_req
	"firewall.setrules"  --Firewall_set_req
	"process.stop" --Process_stop_req
	"rcs.ping" --Rcs_HeartBeat_Req
*/
//define all atomic request structs in here
type File_push_req struct { //only support single file  'file.push'
	Sfileurl, Sfilemd5 string
	DstPath            string
}
type File_pull_req struct { //only support single file   'file.pull'
	Sfilepath string
	Dstdir    string
}
type File_cp_req struct { //file.cp
	Sfilepath string //recursive if it`s a directory,
	Dfilepath string //if exists,overwrite
	Wodir     bool   //only used when Sfilepath is a directory,false:copy the whole directory and it`s bellows,true:only copy the directory`s bellows
}
type File_del_req struct { //file.del
	Sfilepath string //recursive if it`s a directory
	Wobak     bool   //without backup,false:with backup,true:without backup
}
type File_rename_req struct {
	Sfilepath string
	Newname   string
}
type File_grep_req struct { //file.grep
	Sfilepath  string
	Patternstr string //regular expression
}
type File_replace_req struct { //file.replace
	Sfilepath  string //must be single file,replace the whole file
	Patternstr string //regular expression
	Repltext   string
}
type File_mreplace_req struct { //file.mreplace
	Sfiledir           string //specify directory
	Filenamepatternstr string //specify the filename regular expression in the 'Sfiledir' field
	Patternstr         string //regular expression
	Repltext           string
}
type File_md5sum_req struct { //file.md5sum
	Sfilepath string //if directory,compute the md5sum of all the files in the directory
}
type File_ckmd5sum_req struct { //similar to md5sum -c md5file file.ckmd5sum
	Md5filepath string
}
type File_zip_req struct {
	Sfilepath   string // is a file or directory
	Zipfilepath string // like abc.zip  d:\aaa.zip
}
type File_unzip_req struct {
	Zipfilepath string //specify the zip filepath
	Dstdir      string //where unzip to,if not specified,same as the Zipfilepath`s dir
	Wdir        bool
}
type File_gzip_req struct {
	Sfilepath string // is a file or directory
	Dstdir    string //if not specified,same as the Sfilepath`s dir
}
type File_gunzip_req struct {
	Zipfilepath string //specify the zip filepath
	Dstdir      string //where unzip to,if not specified,same as the Zipfilepath`s dir
}
type File_tar_req struct {
	Sfilepath string // is a file or directory
	Dstdir    string //if not specified,same as the Sfilepath`s dir
}
type File_untar_req struct {
	Zipfilepath string //specify the zip filepath
	Dstdir      string //where unzip to,if not specified,same as the Zipfilepath`s dir
}

//==============================================
type Cmd_run_req struct { //cmd.run
	Cmd     string
	CmdArgs []string
}
type Cmd_script_req struct { //cmd.script
	FileUrl    string
	FileMd5    string
	ScriptArgs []string
	Stype      string // bat,shell,py,pws,perl
}

//==============================================

type Os_restart_req struct { //os.restart
	Delay       bool
	Delaysecond int64
}
type Os_shutdown_req struct { //os.shutdown
	Delay       bool
	Delaysecond int64
}
type Os_setpwd_req struct { //os.setpwd
	Username string
	Passwd   string
}

//==============================================
type Firewall_set_req struct { //firewall.setrules
	Rulename []string
	Op       string //at present,just support the code defined in 'const'
}

//==============================================
type Process_stop_req struct { //process.stop
	Imagename []string
	Doforce   bool
}

//==============================================
type Rcs_ping_req struct { //rcs.ping
}

//==============================================
type Atomicrequest interface { //indicate an atomic request
	Handle(*Atomicresponse) error
}

type Atomicresponse struct { //indicate an atomic response
	Flag   bool
	Result string
}
