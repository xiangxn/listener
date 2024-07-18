#!/bin/bash
delfile(){
    if [ -z "$2" ]; then
        if [ -f "$1" ]; then
            rm $1
        fi
    else
        if [ -d "$1" ]; then
            rm -rf $1
        fi
    fi    
}

echo "Clear Temporary Files..."
delfile ./listener
delfile ./Trader.abi
delfile ./Trader.bin
delfile ./trader/Trader.go
delfile ./databackup 1

echo "Clear contract compilation cache..."
forge clean