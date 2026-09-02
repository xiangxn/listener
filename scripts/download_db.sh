#! /bin/bash
# 从服务器拉回 MongoDB 备份(databackup/) 并恢复到本地 MongoDB
# 用法: scripts/download_db.sh [DB_NAME, 默认 ethlistener]
# 依赖 sshpass + mongorestore(docker exec listener-mongodb 中执行)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

DB_NAME="ethlistener"
if [ $# -gt 0 ]; then
    DB_NAME=$1
fi
echo -n "IP: "
read ip
read -s -p "Password: " pw
echo
rm -rf ./databackup
sshpass -p $pw rsync -avz --progress root@$ip:/root/databackup/ ./databackup/

mongorestore --db ${DB_NAME} --drop ./databackup/${DB_NAME}

# mongodump -d ethlistener -o ./databackup
