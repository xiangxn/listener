#!/bin/bash

. clean.sh

echo "Build contract..."
forge build


GOPATH=~/go/bin
# 合约名称
CONTRACT_NAME="Trader"
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
$GOPATH/abigen --abi="${CONTRACT_NAME}.abi" --bin="${CONTRACT_NAME}.bin" --pkg=trader --out="trader/${CONTRACT_NAME}.go"

echo "Go binding file ${CONTRACT_NAME}.go has been generated."
