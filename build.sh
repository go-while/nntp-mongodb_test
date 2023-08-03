#!/bin/bash
PATH="$PATH:/usr/local/go/bin"
export GOPATH=$(pwd)
export GO111MODULE=auto
go vet mongodbtest.go || exit 1
go fmt mongodbtest.go
go build mongodbtest.go
RET=$?
echo $(date)
test $RET -gt 0 && echo "BUILD FAILED! RET=$RET" || echo "BUILD OK!"
exit $RET
