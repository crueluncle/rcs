package modules

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//内置的执行模块
type File struct { //File模块,对文件或目录的操作,方法:Push,Pull,Cp,Del,Grep,Replace,Mreplace,Md5sum,Ckmd5sum
}
type File_push_req struct { //只支持文件
	Sfileurl, Sfilemd5 string
	DstPath            string
}
type File_pull_req struct { //只支持文件
	Sfilepath string
	Dstdir    string
}
type File_cp_req struct { //是目录则默认递归,存在则默认覆盖
	Sfilepath string
	Dfilepath string
	Wodir     bool //目录复制时生效,是否不带Sfilepath中最后的目录名,false:带，true:不带
}
type File_del_req struct { //是目录则默认递归
	Sfilepath string
	Wobak     bool //without bak 是否不创建备份
}
type File_grep_req struct {
	Sfilepath  string
	Patternstr string //正则表达式
}
type File_replace_req struct { //单文件,正则表达式全文替换
	Sfilepath  string
	Patternstr string //正则表达式
	Repltext   string
}
type File_mreplace_req struct { //指定目录下,匹配指定文件名规则的所有文件,做全文替换,创建备份
	Sfiledir           string
	Filenamepatternstr string
	Patternstr         string //正则表达式
	Repltext           string
}
type File_md5sum_req struct { //是目录,则计算目录下所有文件的md5
	Sfilepath string
}
type File_ckmd5sum_req struct { //similar to md5sum -c md5file
	Md5filepath string
}

func (f File) Push(seb File_push_req, res *Atomicresponse) error {
	//log.Println("handle 1 request:File_Push_Req ", seb)
	if err := Downloadfilefromurl(seb.Sfileurl, seb.Sfilemd5, seb.DstPath); err != nil {
		//log.Println("downloadfilefromjobsvr:", err)
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = seb.Sfilemd5
	return nil
}
func (f File) Pull(seb File_pull_req, res *Atomicresponse) error {
	return nil
}
func (f File) Cp(seb File_cp_req, res *Atomicresponse) error {
	err := Cpall(seb.Sfilepath, seb.Dfilepath, seb.Wodir)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
	return nil
}
func (f File) Del(seb File_del_req, res *Atomicresponse) error {
	if seb.Wobak { //without bak
		err := os.RemoveAll(seb.Sfilepath)
		if err != nil {
			res.Flag = false
			res.Result = err.Error()
			return err
		}
		res.Flag = true
		res.Result = "success!"
	} else { //with bak
		t := time.Now().Unix()
		dfilepath := seb.Sfilepath + `-bk` + strconv.FormatInt(t, 10)
		/*//bakup first
		err := Cpall(seb.Sfilepath, dfilepath, true)
		if err != nil {
			res.Flag = false
			res.Result = err.Error()
			return err
		}
		//then delete
		err = os.RemoveAll(seb.Sfilepath)
		if err != nil {
			res.Flag = false
			res.Result = err.Error()
			return err
		}*/
		err := os.Rename(seb.Sfilepath, dfilepath) //call os.rename for backup and delete
		if err != nil {
			res.Flag = false
			res.Result = err.Error()
			return err
		}
		res.Flag = true
		res.Result = "success,backup in " + dfilepath

	}
	return nil
}
func (f File) Grep(seb File_grep_req, res *Atomicresponse) error {
	fd, err := os.Open(seb.Sfilepath)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	defer fd.Close()
	//rx := regexp.MustCompile(seb.Patternstr)
	bufrd := bufio.NewReader(fd)
	var linestr string
	var rs bool
	for err != io.EOF {
		linestr, err = bufrd.ReadString('\n')
		if rs, _ = regexp.MatchString(seb.Patternstr, linestr); rs {
			res.Result += linestr
		}
	}
	res.Flag = true
	return nil
}
func (f File) Replace(seb File_replace_req, res *Atomicresponse) error {
	//ioutil.ReadFile read the hole content to  memory once,that`s a risk point for a 'huge file'
	fi, err := os.Stat(seb.Sfilepath)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	content, err := ioutil.ReadFile(seb.Sfilepath)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	rx := regexp.MustCompile(seb.Patternstr)
	if !rx.Match(content) || seb.Repltext == seb.Patternstr {
		res.Flag = true
		res.Result = seb.Sfilepath + `  ` + "Nochanged\n"
		return nil
	}
	//content = rx.ReplaceAll(content, []byte(seb.Repltext))
	content = rx.ReplaceAllLiteral(content, []byte(seb.Repltext))
	if err := ioutil.WriteFile(seb.Sfilepath, content, fi.Mode()); err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = seb.Sfilepath + `  ` + "Changed\n"
	return nil
}
func (f File) Mreplace(seb File_mreplace_req, res *Atomicresponse) error {
	err, files := Listmatchfiles(seb.Sfiledir, seb.Filenamepatternstr)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	req := new(File_replace_req)
	req.Patternstr = seb.Patternstr
	req.Repltext = seb.Repltext
	eachres := new(Atomicresponse)

	if len(files) == 0 {
		res.Flag = true
		res.Result = "No matched files"
	}

	for _, file := range files {
		req.Sfilepath = file
		if err := f.Replace(*req, eachres); err != nil { //可能部分成功,需输出信息
			return err
		}
		res.Result += eachres.Result
	}
	res.Flag = true
	return nil
}
func (f File) Md5sum(seb File_md5sum_req, res *Atomicresponse) error {
	// output format : RWOSFR2FFSDFADF898DF:::/tmp/test/sdf.ini
	ex, dr, err := Isexistdir(seb.Sfilepath)
	if !ex {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	if !dr {
		md5s, _ := FileMd5(seb.Sfilepath)
		res.Flag = true
		res.Result = md5s + `:::` + seb.Sfilepath
	} else {
		wf := func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}
			md5s, err := FileMd5(path)
			if err != nil {
				return err
			}
			res.Flag = true
			res.Result += md5s + `:::` + path + "\n"
			return nil
		}
		if err := filepath.Walk(seb.Sfilepath, wf); err != nil {
			res.Flag = false
			res.Result = err.Error()
			return err
		}
	}
	return nil
}
func (f File) Ckmd5sum(seb File_ckmd5sum_req, res *Atomicresponse) error {
	/* the md5file format :
	RWOSFR2FFSDFADF898DF:::/tmp/test/sdf.ini
	RWOSFR2FFSDFADF898DF:::/tmp/test/set.sh
	*/
	fd, err := os.Open(seb.Md5filepath)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	defer fd.Close()
	bufrd := bufio.NewReader(fd)
	entry := make([]string, 2)
	var linestr, md5s string
	var RIGHT, WRONG, ERR int

	for err != io.EOF {
		linestr, err = bufrd.ReadString('\n')
		//windows file line end with '\r\n';unix-like file line end with '\n',so should trim '\n' and '\r' by step
		linestr = strings.TrimSuffix(linestr, "\n")
		linestr = strings.TrimSuffix(linestr, "\r")
		entry = strings.Split(linestr, `:::`)
		if len(entry) != 2 { //filter black line and wrong format line
			continue
		}
		md5s, err = FileMd5(entry[1])
		if err == nil {
			if md5s == entry[0] {
				res.Result += entry[1] + `:::CHECK RIGHT` + "\n"
				RIGHT++
			} else {
				res.Result += entry[1] + `:::CHECK WRONG` + "\n"
				WRONG++
			}
		} else {
			res.Result += entry[1] + `:::` + err.Error() + "\n"
			ERR++
		}
	}
	res.Flag = true
	res.Result += fmt.Sprintf("------Statistics,RIGHT:%d,WRONG:%d,ERROR:%d------", RIGHT, WRONG, ERR)
	return nil
}
