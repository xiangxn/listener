#!/bin/bash

# 用 docker 启动 MongoDB (替代已卸载的本地 mongod)
# 数据持久化在当前项目 data/mongodb 目录
# 停止容器后数据保留在宿主机目录, 重新启动/删除容器都不丢

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
data_dir=${ROOT_DIR}/data/mongodb
container_name=listener-mongodb

mkdir -p ${data_dir}

# 幂等: 容器已存在则直接启动
if docker ps -a --format '{{.Names}}' | grep -q "^${container_name}$"; then
    docker start ${container_name}
    exit 0
fi

docker run -d \
    --name ${container_name} \
    -p 127.0.0.1:27017:27017 \
    -v ${data_dir}:/data/db \
    --restart unless-stopped \
    mongo:7.0
