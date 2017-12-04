#!/bin/bash

(cd rcsagent/unix/ && go build rcsagent.go)
(cd rcscli/ && go build rcscli.go)
(cd rcsfileregistry/ && go build rcsfileregistry.go)
(cd rcsjobsvr/ && go build rcsjobsvr.go)
(cd rcsmaster/ && go build rcsmaster.go)
(cd rcsqueryapi/ && go build rcsqueryapi.go)

mkdir -p rcs_release/rcsagent/cfg
mkdir -p rcs_release/rcsagent/log
mkdir -p rcs_release/rcscli/cfg
mkdir -p rcs_release/rcscli/log
mkdir -p rcs_release/rcsfileregistry/cfg
mkdir -p rcs_release/rcsfileregistry/log
mkdir -p rcs_release/rcsjobsvr/cfg
mkdir -p rcs_release/rcsjobsvr/log
mkdir -p rcs_release/rcsmaster/cfg
mkdir -p rcs_release/rcsmaster/log
mkdir -p rcs_release/rcsqueryapi/log
mkdir -p rcs_release/rcsqueryapi/cfg


cp -f rcsagent/unix/rcsagent rcs_release/rcsagent/
cp -f rcscli/rcscli rcs_release/rcscli
cp -f rcsfileregistry/rcsfileregistry rcs_release/rcsfileregistry
cp -f rcsjobsvr/rcsjobsvr rcs_release/rcsjobsvr
cp -f rcsmaster/rcsmaster rcs_release/rcsmaster
cp -f rcsqueryapi/rcsqueryapi rcs_release/rcsqueryapi

tar -zcf  rcs_release.tgz rcs_release	
rm -rf rcs_release
