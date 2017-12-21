package main

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	ss := `D:\Install-bk\Install.zip`
	//dd := `c:\xxxx`
	log.Println(unzip(ss, "", true))
}
func unzip(srczipfile, dstdir string, Wdir bool) error {
	if dstdir == "" {
		dstdir = filepath.Dir(srczipfile)
	}
	var dest string
	if Wdir {
		dest = filepath.Join(dstdir, strings.TrimSuffix(filepath.Base(srczipfile), filepath.Ext(srczipfile)))
	} else {
		dest = dstdir
	}

	unzip_file, err := zip.OpenReader(srczipfile)
	if err != nil {
		return err
	}
	os.MkdirAll(dest, 0755)
	for _, f := range unzip_file.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			_, err = io.Copy(f, rc)
			if err != nil {
				if err != io.EOF {
					return err
				}
			}
			f.Close()
		}
	}
	unzip_file.Close()
	return nil
}
