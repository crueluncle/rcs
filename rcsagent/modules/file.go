package modules

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

//内置的执行模块
type File struct { //File模块,对文件或目录的操作,方法:Push,Pull,Cp,Del,Grep,Replace,Mreplace,Md5sum
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
	Wodir     bool //目录复制时,是否带Sfilepath中最后的目录名,false:带，true:不带
}
type File_del_req struct { //是目录则默认递归,创建备份
	Sfilepath string
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

func (f File) Push(seb File_push_req, res *Atomicresponse) error {
	log.Println("handle 1 request:File_Push_Req ", seb)
	if err := Downloadfilefromurl(seb.Sfileurl, seb.Sfilemd5, seb.DstPath); err != nil {
		log.Println("downloadfilefromjobsvr:", err)
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = seb.Sfilemd5
	return nil
}
func (f File) Pull(seb File_pull_req, res *Atomicresponse) error { return nil }
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
	err := os.RemoveAll(seb.Sfilepath)
	if err != nil {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	res.Flag = true
	res.Result = "success!"
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
	bufrd := bufio.NewReader(fd)
	for {
		linestr, err := bufrd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if rs, _ := regexp.MatchString(seb.Patternstr, linestr); rs {
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
	if !rx.Match(content) {
		res.Flag = true
		res.Result = seb.Sfilepath + `  ` + "Nochanged\n"
		return nil
	}
	//content = rx.ReplaceAll(content, []byte(seb.Repltext))
	content = rx.ReplaceAllLiteral(content, []byte(seb.Repltext))
	//newcontent := rx.ReplaceAllString(string(content), seb.repltext)
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
		res.Result = "No files matched"
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
	ex, dr, err := Isexistdir(seb.Sfilepath)
	if !ex {
		res.Flag = false
		res.Result = err.Error()
		return err
	}
	if !dr {
		md5s, _ := FileMd5(seb.Sfilepath)
		res.Flag = true
		res.Result = md5s + `  ` + seb.Sfilepath //md5sum -c file的标准格式
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
			res.Result += md5s + `  ` + path + "\n"
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
