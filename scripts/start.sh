#!/bin/bash
repository=$1
ref=$2
commit_sha=$3

CONTAINER_NAME=memezis-bot
IMAGE_NAME=docker.pkg.github.com/cherya/memezis-bot/memezis-bot:latest

msg="Updating *${repository}* \nref: *${ref}* \ncommit: (${commit_sha})[https://github.com/cherya/memezis-bot/commit/${commit_sha}]"
bash scripts/notify.sh "$msg"

docker pull $IMAGE_NAME
docker stop $CONTAINER_NAME && docker rm $CONTAINER_NAME
docker run --mount type=bind,source=$(pwd)/production.env,target=/app/production.env --name $CONTAINER_NAME -d --net=host -d $IMAGE_NAME

sleep 5

if [ ! "$(docker ps -q -f name=$CONTAINER_NAME)" ]; then
  msg="Update failed for *${repository}* \nref: *${ref}* \ncommit: (${commit_sha})[https://github.com/cherya/memezis-bot/commit/${commit_sha}]"
  bash scripts/notify.sh "$msg"
else
  msg="Successfully updated *${repository}* \nref: *${ref}* \ncommit: [${commit_sha}](https://github.com/cherya/memezis-bot/commit/${commit_sha})"
  bash scripts/notify.sh "$msg"
fi