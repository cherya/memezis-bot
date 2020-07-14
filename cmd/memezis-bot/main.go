package main

import (
	"context"
	"flag"
	"github.com/cherya/memezis-bot/internal/dailyword"

	"github.com/cherya/memezis-bot/internal/banhammer"
	"github.com/cherya/memezis-bot/internal/bot"
	"github.com/cherya/memezis-bot/internal/config"
	"github.com/cherya/memezis-bot/internal/logger"

	"github.com/cherya/memezis/pkg/memezis"
	"github.com/cherya/memezis/pkg/queue"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const logDateFormat = "02-01-2006 15:04:05"

func main() {
	logger.Init(log.DebugLevel, logDateFormat)

	initEnv()

	var redisPool = &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial(
				"tcp",
				config.GetValue(config.RedisAddress),
				redis.DialPassword(config.GetValue(config.RedisPassword)),
			)
		},
	}

	memezisConn, err := grpc.Dial(
		config.GetValue(config.MemezisAddress),
		//grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		grpc.WithPerRPCCredentials(tokenAuth{token: config.GetValue(config.MemezisToken)}),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatal("can't dial memezis", err)
	}
	defer memezisConn.Close()

	memezisClient := memezis.NewMemezisClient(memezisConn)

	bbot, err := bot.NewBot(
		config.GetValue(config.TgBotToken),
		queue.NewManager(redisPool, "memezis"),
		memezisClient,
		dailyword.NewWordGenerator(redisPool),
		banhammer.NewBanHammer(redisPool, 300),
		config.GetInt64(config.PublicationChannelId),
		config.GetInt64(config.SuggestionChannelId),
		config.GetInt(config.OwnerID),
	)

	if err != nil {
		log.Fatalf("Bot connection error", err)
		return
	}

	err = bbot.Start()

	if err != nil {
		log.Fatal("Bot internal error:", err)
		return
	}
}

type tokenAuth struct {
	token string
}

// Return value is mapped to request headers.
func (t tokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return false
}

func initEnv() {
	env := flag.String("env", "local.env", "env file with config values")
	flag.Parse()
	log.Infof("Loading env from %s", *env)
	err := godotenv.Load(*env)

	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	if config.GetBool(config.Debug) {
		logEnv(env)
	}
}

func logEnv(env *string) {
	envMap, _ := godotenv.Read(".env", *env)
	for key, val := range envMap {
		log.Infof("%s = %s", key, val)
	}
}
