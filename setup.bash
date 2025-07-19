#!/bin/bash
[ $(which wget) == "" ] && echo "you need wget for this installer to work" && exit 1

rm go-discover
rm run-discover.bash

wget https://github.com/savildatech/go-discover/raw/refs/heads/main/go-discover
wget https://github.com/savildatech/go-discover/raw/refs/heads/main/run-discover.bash

chmod +x go-discover
chmod +x run-discover
