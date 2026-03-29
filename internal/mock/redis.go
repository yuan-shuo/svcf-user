package mock

import (
	"user/internal/errs"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// IsCodeError 检查错误是否为指定的错误码
func IsCodeError(err error, code int) bool {
	if err == nil {
		return false
	}
	if e, ok := errs.IsCodeError(err); ok {
		return e.Code == code
	}
	return false
}

// SetupTestRedis 创建测试用的 Redis 实例
func SetupTestRedis(t miniredis.Tester) (*redis.Redis, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	conf := redis.RedisConf{
		Host: s.Addr(),
		Type: "node",
	}
	r := redis.MustNewRedis(conf)
	return r, s
}
