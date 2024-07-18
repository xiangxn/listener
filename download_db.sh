#! /bin/bash
echo -n "IP: "
read ip
read -s -p "Password: " pw
echo
rm -rf ./databackup
sshpass -p $pw rsync -avz --progress root@$ip:/root/databackup/ ./databackup/

mongorestore --db ethlistener --drop ./databackup/ethlistener

# mongodump -d ethlistener -o ./databackup