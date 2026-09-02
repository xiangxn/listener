#!/bin/bash
# 清理仓库根目录下构建/运行生成的临时文件:
#   ./listener 二进制、Trader.abi/.bin、trader/Trader.go、databackup、logs/、编译缓存(out/cache)
# 不删除: data/mongodb(本地 MongoDB 数据)、data/*.json(黑名单, git 跟踪)
# 从任意目录执行均可(自动切到仓库根目录); BuildTraderToGo.sh 也会 source 本脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

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
delfile ./*.abi
delfile ./*.bin
delfile ./trader/Trader.go
delfile ./databackup 1
delfile ./logs 1

if command -v forge >/dev/null 2>&1; then
    echo "Clear contract compilation cache..."
    forge clean
else
    echo "Skip forge clean (forge not installed)"
fi
