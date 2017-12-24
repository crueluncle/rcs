package main

import (
	"archive/zip"
	//	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	if err := compress(`D:\QQVipDownload\LOL_V4.0.5.5_FULL.7z.001`, `D:\QQVipDownload\aaaa.zip`); err != nil {
		fmt.Println(err)
	}
}

// 参数frm可以是文件或目录，不会给dst添加.zip扩展名
func compress(sourcepath, zipfilename string) error {
	buf := bytes.NewBuffer(make([]byte, 0, 10*1024*1024)) // 创建一个读写缓冲
	zipwriter := zip.NewWriter(buf)                       // 用压缩器包装该缓冲
	// 用Walk方法来将所有目录下的文件写入zipwriter
	wf := func(path string, info os.FileInfo, err error) error {
		var filecontent []byte
		if err != nil {
			return filepath.SkipDir
		}
		header, err := zip.FileInfoHeader(info) // 转换为zip格式的文件信息
		if err != nil {
			return filepath.SkipDir
		}
		header.Name, _ = filepath.Rel(filepath.Dir(sourcepath), path)
		if !info.IsDir() {
			// 确定采用的压缩算法（这个是内建注册的deflate）
			header.Method = 8
			filecontent, err = ioutil.ReadFile(path) // 获取文件内容,内存占用高风险,改为bufio
			if err != nil {
				return filepath.SkipDir
			}
		} else {
			filecontent = nil
		}
		// 上面的部分如果出错都返回filepath.SkipDir
		// 下面的部分如果出错都直接返回该错误
		// 目的是尽可能的压缩目录下的文件，同时保证zip文件格式正确
		w, err := zipwriter.CreateHeader(header) // 创建一条记录并写入文件信息
		if err != nil {
			return err
		}
		_, err = w.Write(filecontent) // 非目录文件会写入数据，目录不会写入数据
		if err != nil {               // 因为目录的内容可能会修改
			return err // 最关键的是我不知道咋获得目录文件的内容
		}
		return nil
	}

	err := filepath.Walk(sourcepath, wf)
	if err != nil {
		return err
	}
	zipwriter.Close()                      // 关闭压缩器，让压缩器缓冲中的数据写入buf
	zipfile, err := os.Create(zipfilename) // 建立zip文件
	if err != nil {
		return err
	}
	defer zipfile.Close()
	_, err = buf.WriteTo(zipfile) // 将buf中的数据写入文件
	if err != nil {
		return err
	}
	return nil
}
