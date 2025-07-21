#!/bin/bash

execs=("broadcast-svc" "listen-udp")

for exec in "${execs[@]}"; do
    rm "$exec"
    wget "https://github.com/savildatech/go-discover/raw/refs/heads/main/bin/$exec"
    chmod +x "$exec"
done