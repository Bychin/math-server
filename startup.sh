#!/bin/bash

server_path=./
server_src=(main.go server.go)
server_bin=server

go build -o $server_bin ${server_src[0]} ${server_src[1]}

screen -d -m $server_path/$server_bin
