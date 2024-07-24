#! /bin/bash
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