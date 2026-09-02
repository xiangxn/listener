#!/bin/bash
# 编译合约并生成 Go binding(trader/Trader.go)
# 用法: scripts/BuildTraderToGo.sh [合约名, 默认 Trader]
# 依赖: forge(编译合约)、jq(提取 abi/bin)、abigen(生成 Go 代码, 需先 go install github.com/ethereum/go-ethereum/cmd/abigen@latest)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

. "$SCRIPT_DIR/clean.sh"

echo "Build contract..."
forge build


GOPATH=~/go/bin
# 合约名称
CONTRACT_NAME="Trader"
if [ $# -gt 0 ]; then
    CONTRACT_NAME=$1
fi
CONTRACT_JSON="trader/out/${CONTRACT_NAME}.sol/${CONTRACT_NAME}.json"

# 检查文件是否存在
if [ ! -f "$CONTRACT_JSON" ]; then
    echo "File $CONTRACT_JSON does not exist!"
    exit 1
fi

# 提取ABI
ABI=$(jq -c '.abi' "$CONTRACT_JSON")
echo "$ABI" > "${CONTRACT_NAME}.abi"

# 提取字节码
BYTECODE=$(jq -r '.bytecode.object' "$CONTRACT_JSON")
echo "$BYTECODE" > "${CONTRACT_NAME}.bin"

# 使用abigen生成Go文件
$GOPATH/abigen --abi="${CONTRACT_NAME}.abi" --bin="${CONTRACT_NAME}.bin" --pkg=trader --out="trader/Trader.go"

echo "Go binding file Trader.go has been generated."
