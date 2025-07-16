#!/bin/bash
[ $(which wget) == "" ] && echo "you need wget for this installer to work" && exit 1

rm go-discover
wget https://github.com/savildatech/go-discover/raw/refs/heads/main/go-discover
chmod +x go-discover