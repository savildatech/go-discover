#!/bin/bash

paths=("./src/broadcast-svc" "./src/listen-udp")

rpaths=()

for path in "${paths[@]}"; do
    [ ! -d "$path" ] && echo "$path not found" && continue
    rpaths+=("$(realpath $path)")
done

rm ./bin/*

for p in "${rpaths[@]}"; do
    cd "$p" && ./build.bash
done