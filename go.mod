module github.com/cherya/memezis-bot

require (
	github.com/cherya/memezis/pkg/memezis v0.0.0-20200717155514-9c44f874904c
	github.com/cherya/memezis/pkg/queue v0.0.0-20200704133524-636b22b379bd
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.0.0-rc1
	github.com/gocraft/work v0.5.1 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/joho/godotenv v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron v1.2.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/tidwall/gjson v1.6.0
	google.golang.org/grpc v1.30.0
)

go 1.13
