#!/bin/bash
repository=$1
ref=$2
commit_sha=$3

CONTAINER_NAME=memezis-bot
IMAGE_NAME=docker.pkg.github.com/cherya/memezis-bot/memezis-bot:latest

docker pull docker.pkg.github.com/cherya/memezis-bot/memezis-bot:latest
docker stop memezis-bot && docker rm memezis-bot
docker run --mount type=bind,source=$(pwd)/production.env,target=/app/production.env --name $CONTAINER_NAME -d --net=host -d $IMAGE_NAME

if [ ! "$(docker ps -q -f name=$CONTAINER_NAME)" ]; then
  msg="Update failed for *${repository}* \n ref: *${ref}* \n commit: (${commit_sha})[https://github.com/cherya/memezis-bot/commit/${commit_sha}]"
  bash scripts/notify.sh "$msg"
else
  # shellcheck disable=SC2082
  msg="Successfully updated *${repository}* \n ref: *${ref}* \n commit: [${commit_sha}](https://github.com/cherya/memezis-bot/commit/${commit_sha})"
  bash scripts/notify.sh "$msg"
fi