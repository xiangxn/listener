#! /bin/bash
# 从服务器拉回配置文件 + 黑名单 json(远端 /root/listener/{config,data}/ → 本地仓库根目录/data/)
# 用法: scripts/download_config.sh [NET_NAME, 默认 eth]
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
mkdir -p ./data
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/${NET_NAME}.config.yaml ./${NET_NAME}.config.yaml
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/data/${NET_NAME}_token_blacklist.json ./data/${NET_NAME}_token_blacklist.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/data/${NET_NAME}_pool_blacklist.json ./data/${NET_NAME}_pool_blacklist.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/data/${NET_NAME}_token_erc20a.json ./data/${NET_NAME}_token_erc20a.json
