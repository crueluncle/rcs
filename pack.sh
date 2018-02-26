#!/bin/bash
basedir="rcs_release"
for i in `find . -type f -name "main.go"`
do 
	fdir=`dirname $i`
	lastelem=`echo "$fdir" |awk -F "/" '{print $NF}'`
	go build  -o $basedir/$fdir/$lastelem $i
done

tar -zcf  $basedir.tgz $basedir
rm -rf $basedir 
mv -f $basedir.tgz /root/
