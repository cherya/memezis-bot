docker pull docker.pkg.github.com/cherya/memezis-bot/memezis-bot:latest
docker stop memezis-bot
docker run --mount type=bind,source=.production.env,target=/app/.production.env -d --net=host -d memezis-bot:latest