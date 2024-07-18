#! /bin/bash
echo -n "IP: "
read ip
read -s -p "Password: " pw
echo
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/config.json ./config.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/token_blacklist.json ./token_blacklist.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/pool_blacklist.json ./pool_blacklist.json
sshpass -p $pw rsync -avz --progress root@$ip:/root/listener/token_erc20a.json ./token_erc20a.json