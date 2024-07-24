#! /bin/bash

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

sshpass -p $pw rsync -avz --progress ./${NET_NAME}_token_blacklist.json root@$ip:/root/listener/${NET_NAME}_token_blacklist.json
sshpass -p $pw rsync -avz --progress ./${NET_NAME}_pool_blacklist.json root@$ip:/root/listener/${NET_NAME}_pool_blacklist.json
sshpass -p $pw rsync -avz --progress ./${NET_NAME}_token_erc20a.json root@$ip:/root/listener/${NET_NAME}_token_erc20a.json