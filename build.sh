#!/bin/bash
export GOPATH=$HOME/go
if [ "${GITHUB_ACTIONS}" == "true" ]; then
go test 1> debug.out
else
go test -v
fi

go get 1>> debug.out

if [ $? == 0 ]; then
  if [ "${GOOS}" == "windows" ]; then
    if [ "${GITHUB_ACTIONS}" == "true" ]; then 
go build -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle.exe *.go 1>> debug.out
echo 'noodle.exe'
    else
go build -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle.exe *.go    
    fi
  else
    if [ "${GITHUB_ACTIONS}" == "true" ]; then 
go build -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle *.go 1>> debug.out
echo 'noodle'
    else
go build -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle *.go
    fi
  fi
fi
