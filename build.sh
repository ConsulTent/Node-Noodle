#!/bin/bash
export GOPATH=$HOME/go
go build -i -v -ldflags="-X main.gitver=$(git describe --always --long --dirty)" -o noodle *.go
