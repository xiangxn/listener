#!/bin/bash

if [ $# -gt 0 ]; then
bash BuildTraderToGo.sh $1
else
. BuildTraderToGo.sh
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .