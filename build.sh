#!/bin/bash
export GOPATH=$HOME/go
go test
if [ $? == 0 ]; then
  if [ "${GOOS}" == "windows" ]; then
go build -i -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle.exe *.go
echo 'noodle.exe'
  else
go build -i -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle *.go
echo 'noodle'
  fi
fi
