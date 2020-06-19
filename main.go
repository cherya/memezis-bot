package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cherya/memezis-bot/bot"
	"github.com/cherya/memezis-bot/config"
	"github.com/cherya/memezis-bot/memezis_client"

	"github.com/cherya/memezis/pkg/queue"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

func initEnv() {
	env := flag.String("env", "local.env", "env file with config values")
	flag.Parse()
	log.Printf("Loading env from %s", *env)
	err := godotenv.Load(*env)

	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	logEnv(env)
}

func logEnv(env *string) {
	envMap, _ := godotenv.Read(".env", *env)
	for key, val := range envMap {
		fmt.Printf("[godotenv] %s = %s\n", key, val)
	}
}

func main() {
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

	bbot, err := bot.NewBot(
		config.GetValue(config.TgBotToken),
		queue.NewManager(redisPool),
		memezis_client.NewClient(
			config.GetValue(config.MemezisAddress),
			config.GetValue(config.MemezisToken),
		),
		NewBanHammer(redisPool, 300),
		config.GetInt64(config.PublicationChannelId),
		config.GetInt64(config.SuggestionChannelId),
		config.GetInt(config.OwnerID),
	)

	if err != nil {
		log.Println("Bot connection error:", err)
		return
	}

	err = bbot.Start()

	if err != nil {
		log.Println("Bot internal error:", err)
		return
	}
}

type BanHammer struct {
	redis      *redis.Pool
	banSeconds int
}

func NewBanHammer(r *redis.Pool, banSeconds int) *BanHammer {
	return &BanHammer{
		banSeconds: banSeconds,
		redis:      r,
	}
}

func (bh *BanHammer) banKey(u string) string {
	return fmt.Sprintf("ban:%s", u)
}

func (bh *BanHammer) Ban(u string) error {
	conn := bh.redis.Get()
	defer conn.Close()

	_, err := conn.Do("SET", bh.banKey(u), true, "EX", bh.banSeconds)
	return err
}

func (bh *BanHammer) Permaban(u string) error {
	conn := bh.redis.Get()
	defer conn.Close()

	_, err := conn.Do("SET", bh.banKey(u), true)
	return err
}

func (bh *BanHammer) Unban(u string) error {
	conn := bh.redis.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", bh.banKey(u))
	return err
}

func (bh *BanHammer) IsBanned(u string) (bool, error) {
	conn := bh.redis.Get()
	defer conn.Close()

	v, err := redis.Bool(conn.Do("GET", bh.banKey(u)))
	if errors.Cause(err) == redis.ErrNil {
		return false, nil
	}
	return v, err
}
