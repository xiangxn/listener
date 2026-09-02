#! /bin/bash
# 发布到服务器: 编译好的 listener 二进制 + 配置文件 + abi + 黑名单 json(本地 data/ → 远端 /root/listener/data/)
# 用法: scripts/publish.sh [NET_NAME, 默认 eth]
# 依赖 sshpass

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

NET_NAME="eth"
if [ $# -gt 0 ]; then
    NET_NAME=$1
fi

echo -n "IP: "
read ip
read -s -p "Password: " pw
echo
sshpass -p $pw rsync -avz --progress ./listener root@$ip:/root/listener/listener
sshpass -p $pw rsync -avz --progress ./${NET_NAME}.config.yaml root@$ip:/root/listener/${NET_NAME}.config.yaml
sshpass -p $pw rsync -avz --progress -r ./abis/ root@$ip:/root/listener/abis/

# 黑名单放远端 data/ 目录(与本地布局一致, bot 按 data/ 前缀读写); rsync-path 先建目录
sshpass -p $pw rsync -avz --rsync-path="mkdir -p /root/listener/data && rsync" --progress \
    ./data/${NET_NAME}_token_blacklist.json root@$ip:/root/listener/data/
sshpass -p $pw rsync -avz --rsync-path="mkdir -p /root/listener/data && rsync" --progress \
    ./data/${NET_NAME}_pool_blacklist.json root@$ip:/root/listener/data/
sshpass -p $pw rsync -avz --rsync-path="mkdir -p /root/listener/data && rsync" --progress \
    ./data/${NET_NAME}_token_erc20a.json root@$ip:/root/listener/data/
