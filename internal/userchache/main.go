package userchache

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

type UserCache struct {
	redis      *redis.Pool
	banSeconds int
}

func NewUserCache(r *redis.Pool) *UserCache {
	return &UserCache{
		redis:      r,
	}
}

func userNameKey(i int64) string {
	return fmt.Sprintf("user_by_post:name:%d", i)
}

func userIDKey(i int64) string {
	return fmt.Sprintf("user_by_post:id:%d", i)
}

func (bh *UserCache) Set(postId int64, name string, id int) error {
	conn := bh.redis.Get()
	defer conn.Close()

	_, err := conn.Do("SET", userNameKey(postId), name, "EX", 7 * 24 * 60 * 60)
	if err != nil {
		return err
	}
	_, err = conn.Do("SET", userIDKey(postId), id, "EX", 7 * 24 * 60 * 60)
	return err
}

func (bh *UserCache) GetName(postId int64) (string, error) {
	conn := bh.redis.Get()
	defer conn.Close()

	name, err := redis.String(conn.Do("GET", userNameKey(postId)))
	return name, err
}

func (bh *UserCache) GetID(postId int64) (string, error) {
	conn := bh.redis.Get()
	defer conn.Close()

	name, err := redis.String(conn.Do("GET", userIDKey(postId)))
	return name, err
}