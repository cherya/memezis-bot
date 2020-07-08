package banhammer

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

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

