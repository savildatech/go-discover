#!/bin/bash

execs=("setup.bash" "bin/broadcast-svc" "bin/listen-udp")

for exec in "${execs[@]}"; do
    rm "$exec"
    wget "https://github.com/savildatech/go-discover/raw/refs/heads/main/$exec"
    chmod +x "$exec"
done