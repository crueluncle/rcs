package modules //定义共用对象及函数
//定义rpc服务模块
import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func Downloadfilefromurl(srcfileurl, srcfilemd5, dstdir string) error {
	//目标文件名与url中uri一致，若文件存在且md5一致则不会下载
	//	log.Println("srcfileurl:", srcfileurl, "dstdir:", dstdir)
	u, e := url.Parse(srcfileurl)
	if e != nil {
		return e
	}
	//bn := strings.Split(u.RequestURI(), `/`)
	filename := u.Query().Get("rename")
	if filename == "" {
		filename = filepath.Base(u.RequestURI())
		if filename == "" {
			return errors.New("srcfileurl is invalid:" + srcfileurl)
		}
	}
	dstfilepath := filepath.Join(dstdir, filename)
	//log.Println("dstfilepath:", dstfilepath)
	if ex, dr, _ := Isexistdir(dstfilepath); ex && !dr {
		md, err := FileMd5(dstfilepath)
		if err == nil && md == srcfilemd5 {
			return nil
		}
	}
	req, _ := http.NewRequest("GET", strings.Split(srcfileurl, `?`)[0], nil)
	//req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Connection", "close")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		//log.Println(err)
		return err
	}
	if resp.StatusCode != 200 {
		//log.Println(errors.New(resp.Status))
		return errors.New(resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(dstfilepath), 0777); err != nil {
		return err
	}
	f1, e := os.OpenFile(dstfilepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if e != nil {
		return e
	}
	md5h := md5.New()
	_, err = io.Copy(io.MultiWriter(f1, md5h), resp.Body)
	if err != nil {
		return err
	}
	if err = f1.Close(); err != nil {
		return err
	}
	if err = resp.Body.Close(); err != nil {
		return err
	}
	if hex.EncodeToString(md5h.Sum(nil)) == srcfilemd5 {
		return nil
	} else {
		return errors.New("md5sum not matched")
	}
}
func cpfile(sfilepath, dfilepath string) error {
	if err := os.MkdirAll(filepath.Dir(dfilepath), 0777); err != nil {
		return err
	}
	sFile, err := os.Open(sfilepath)
	if err != nil {
		return err
	}
	defer sFile.Close()
	eFile, err := os.Create(dfilepath)
	if err != nil {
		return err
	}
	defer eFile.Close()
	_, err = io.Copy(eFile, sFile)
	if err != nil {
		return err
	}
	err = eFile.Sync()
	eFile.Sync()
	if err != nil {
		return err
	}
	return nil
}
func FileMd5(filepath string) (string, error) {
	file, inerr := os.Open(filepath)
	defer file.Close()
	if inerr == nil {
		md5h := md5.New()
		if _, err := io.Copy(md5h, file); err != nil {
			return "", err
		}
		chksum := hex.EncodeToString(md5h.Sum(nil))
		return chksum, nil
	}
	return "", inerr
}
func Isexistdir(name string) (isexist, isdir bool, err error) { //是否存在,是否为目录
	fi, err := os.Stat(name)
	if err == nil || os.IsExist(err) {
		isexist = true
		isdir = fi.IsDir()
		return isexist, isdir, err
	}
	if os.IsNotExist(err) {
		return false, false, err
	}
	if fi != nil {
		isexist = true
		isdir = fi.IsDir()
		return isexist, isdir, err
	}
	return false, false, err
}
func Listmatchfiles(dirname string, filenamepattern string) (error, []string) { //列出给定目录下，文件名匹配filenamepattern的所有文件
	ex, dr, err := Isexistdir(dirname)
	if !ex {
		return err, nil
	}
	if !dr {
		return errors.New(dirname + " is not a dir"), nil
	}
	filelist := []string{}
	wf := func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if rs, _ := regexp.MatchString(filenamepattern, f.Name()); rs {
			filelist = append(filelist, path)
		}
		return nil
	}
	err = filepath.Walk(dirname, wf)
	if err != nil {
		return err, nil
	}
	return nil, filelist
}

func restartos(delay bool, delaysecond int64) error {
	switch runtime.GOOS {
	case "linux":
		if delay && delaysecond > 0 {
			if output, err := exec.Command("shutdown", "-r", `+`, strconv.FormatInt(delaysecond/60, 10)).CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		} else {
			if output, err := exec.Command("reboot").CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		}
	case "windows":
		if delay && delaysecond > 0 {
			if output, err := exec.Command("shutdown", "/r", "/t", strconv.FormatInt(delaysecond, 10)).CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		} else {
			if output, err := exec.Command("shutdown", "/r", "/f").CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		}
	default:
		return errors.New("unsupport os type")
	}
	return nil
}
func shutdownos(delay bool, delaysecond int64) error {
	switch runtime.GOOS {
	case "linux":
		if delay && delaysecond > 0 {
			if output, err := exec.Command("shutdown", "-h", `+`, strconv.FormatInt(delaysecond/60, 10)).CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		} else {
			if output, err := exec.Command("halt").CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		}
	case "windows":
		if delay && delaysecond > 0 {
			if output, err := exec.Command("shutdown", "/s", "/t", strconv.FormatInt(delaysecond, 10)).CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		} else {
			if output, err := exec.Command("shutdown", "/s", "/f").CombinedOutput(); err != nil {
				return errors.New(string(output) + err.Error())
			}
		}
	default:
		return errors.New("unsupport os type")
	}
	return nil
}
func setpasswd(username, passwd string) error {
	switch runtime.GOOS {
	case "linux":
		//echo password | passwd --stdin  username
		cmd := "echo password | passwd --stdin  username"
		comd := exec.Command("bash", "-c", cmd)
		if output, err := comd.CombinedOutput(); err != nil {
			return errors.New(string(output) + err.Error())
		}
	case "windows":
		//net user username passwd
		comd := exec.Command("net", "user", username, passwd)
		if output, err := comd.CombinedOutput(); err != nil {
			return errors.New(string(output) + err.Error())
		}
	default:
		return errors.New("unsupport os type")
	}
	return nil
}
func setrules(rulenamelist []string, opswitch string) error {
	//netsh advfirewall firewall set rule name=$fwrule new enable=no
	//netsh advfirewall firewall set rule name=$fwrule new enable=yes
	//netsh advfirewall firewall delete rule name=$fwrule
	switch opswitch {
	case "disable":
		for _, name := range rulenamelist {

			cmd := exec.Command("netsh", "advfirewall", "firewall", "set", "rule", "name="+name, "new", "enable=no")
			if output, err := cmd.CombinedOutput(); err != nil {
				return errors.New("In rule: " + name + " error: " + string(output) + err.Error())
			}

		}
	case "enable":
		for _, name := range rulenamelist {
			cmd := exec.Command("netsh", "advfirewall", "firewall", "set", "rule", "name="+name, "new", "enable=yes")
			if output, err := cmd.CombinedOutput(); err != nil {
				return errors.New("In rule: " + name + " error: " + string(output) + err.Error())
			}
		}
	case "del":
		for _, name := range rulenamelist {
			cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name)
			if output, err := cmd.CombinedOutput(); err != nil {
				return errors.New("In rule: " + name + " error: " + string(output) + err.Error())
			}
		}
	default:
		return errors.New("Unsupport operation")
	}
	return nil
}
func stopprocess(Imagename []string, isforce bool) error {
	//taskkill /IM $name
	//taskkill /F /IM $name
	if isforce {
		for _, name := range Imagename {
			cmd := exec.Command("taskkill", `/IM`, name)
			if output, err := cmd.CombinedOutput(); err != nil {
				return errors.New("Process: " + name + " error: " + string(output) + err.Error())
			}
		}
	} else {
		for _, name := range Imagename {
			cmd := exec.Command("taskkill", `/F`, `/IM`, name)
			if output, err := cmd.CombinedOutput(); err != nil {
				return errors.New("Process: " + name + " error: " + string(output) + err.Error())
			}
		}
	}
	return nil
}
