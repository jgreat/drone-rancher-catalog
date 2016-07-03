#!/bin/bash
set -x
rm drone-rancher-catalog
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
go build -v -a -tags netgo
docker build --rm -t leankit/drone-rancher-catalog .
