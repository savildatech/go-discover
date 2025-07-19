#!/bin/bash

export BROADCAST_ADDR="192.168.1.255"
export PORT="12345"
export MIN_INTERVAL="10"
export MAX_INTERVAL="30"
export AVG_SECONDS="60"
export CUSTOM_STRING="example_custom"
export SAMPLE_INTERVAL="5"  # optional

go run main.go