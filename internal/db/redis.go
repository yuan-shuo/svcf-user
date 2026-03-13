package db

import "github.com/zeromicro/go-zero/core/stores/redis"

func NewRedis(conf redis.RedisConf) *redis.Redis {
	return redis.MustNewRedis(conf)
}
