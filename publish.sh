#! /bin/bash
echo -n "IP: "
read ip
read -s -p "Password: " pw
echo
sshpass -p $pw rsync -avz --progress ./listener root@$ip:/root/listener/listener
sshpass -p $pw rsync -avz --progress ./config.json root@$ip:/root/listener/config.json
sshpass -p $pw rsync -avz --progress -r ./abis/ root@$ip:/root/listener/abis/

sshpass -p $pw rsync -avz --progress ./token_blacklist.json root@$ip:/root/listener/token_blacklist.json
sshpass -p $pw rsync -avz --progress ./pool_blacklist.json root@$ip:/root/listener/pool_blacklist.json
sshpass -p $pw rsync -avz --progress ./token_erc20a.json root@$ip:/root/listener/token_erc20a.json