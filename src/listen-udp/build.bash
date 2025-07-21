#!/bin/bash

bin_location="../../bin/listen-udp"

export GOOS=linux
export GOARCH=amd64 
export CGO_ENABLED=0
go mod tidy
go build -o "$bin_location" .
