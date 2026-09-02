#!/bin/bash
# 交叉编译 Linux 版 listener(linux/amd64), 用于部署到服务器
# 用法: scripts/build-linux.sh [合约名]
# 依赖 BuildTraderToGo.sh(会先清临时文件 → forge build → 生成 Trader.go)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"  # go build ./listener 二进制输出到仓库根目录

if [ $# -gt 0 ]; then
bash "$SCRIPT_DIR/BuildTraderToGo.sh" $1
else
. "$SCRIPT_DIR/BuildTraderToGo.sh"
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .
