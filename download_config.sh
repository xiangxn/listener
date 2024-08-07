#! /bin/bash

NET_NAME="eth"
if [ $# -gt 0 ]; then
    NET_NAME=$1
fi

echo -n "IP: "
read ip
read -s -p "Password: " pw
echo
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/${NET_NAME}.config.yaml ./${NET_NAME}.config.yaml
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/${NET_NAME}_token_blacklist.json ./${NET_NAME}_token_blacklist.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/${NET_NAME}_pool_blacklist.json ./${NET_NAME}_pool_blacklist.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/${NET_NAME}_token_erc20a.json ./${NET_NAME}_token_erc20a.json