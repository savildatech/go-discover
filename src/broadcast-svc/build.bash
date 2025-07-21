#!/bin/bash

bin_location="../../bin/broadcast-svc"

export GOOS=linux
export GOARCH=amd64 
export CGO_ENABLED=0
go mod tidy
go build -o "$bin_location" .
