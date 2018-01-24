#!/bin/bash

#(cd rcsagent && go build rcsagent_unix.go)
(cd rcscli/ && go build rcscli.go)
(cd rcsfileregistry/ && go build rcsfileregistry.go)
(cd rcsjobsvr/ && go build rcsjobsvr.go)
(cd rcsmaster/ && go build rcsmaster.go)
(cd rcsqueryapi/ && go build rcsqueryapi.go)


#mkdir -p rcs_release/rcsagent/
mkdir -p rcs_release/rcscli/
mkdir -p rcs_release/rcsfileregistry/
mkdir -p rcs_release/rcsjobsvr/
mkdir -p rcs_release/rcsmaster/
mkdir -p rcs_release/rcsqueryapi/

#mv  rcsagent/rcsagent rcs_release/rcsagent/
mv  rcscli/rcscli rcs_release/rcscli/
mv  rcsfileregistry/rcsfileregistry rcs_release/rcsfileregistry/
mv  rcsjobsvr/rcsjobsvr rcs_release/rcsjobsvr/
mv  rcsmaster/rcsmaster rcs_release/rcsmaster/
mv  rcsqueryapi/rcsqueryapi rcs_release/rcsqueryapi/

tar -zcf  rcs_release.tgz rcs_release	
rm -rf rcs_release
