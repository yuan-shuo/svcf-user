package userutils

import (
	"context"
	"database/sql"
	"time"

	"user/internal/model"
)

// UsersModelInterface 用户模型接口
type UsersModelInterface interface {
	FindOneByEmail(ctx context.Context, email string) (*model.Users, error)
	FindOneBySnowflakeId(ctx context.Context, snowflakeId int64) (*model.Users, error)
	Update(ctx context.Context, data *model.Users) error
	Insert(ctx context.Context, data *model.Users) (sql.Result, error)
}

// RedisInterface Redis客户端接口
type RedisInterface interface {
	SetnxExCtx(ctx context.Context, key, value string, seconds int) (bool, error)
	Ttl(key string) (int, error)
	DelCtx(ctx context.Context, keys ...string) (int, error)
	HsetCtx(ctx context.Context, key, field string, value string) error
	HmsetCtx(ctx context.Context, key string, fields map[string]string) error
	HgetCtx(ctx context.Context, key, field string) (string, error)
	HdelCtx(ctx context.Context, key string, fields ...string) (bool, error)
	ExpireCtx(ctx context.Context, key string, seconds int) error
	ExistsCtx(ctx context.Context, key string) (bool, error)
	HgetallCtx(ctx context.Context, key string) (map[string]string, error)
}

// Pipeliner 定义 Pipeline 接口
type Pipeliner interface {
	HSet(ctx context.Context, key string, values ...interface{}) error
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Exec(ctx context.Context) error
}

// RedisPipeliner 支持 Pipeline 的 Redis 客户端接口
type RedisPipeliner interface {
	RedisInterface
	Pipeline() Pipeliner
}

// MqPusherClientInterface 消息队列推送客户端接口
type MqPusherClientInterface interface {
	Push(ctx context.Context, v string) error
}

// 编译期断言：确保 model.UsersModel 实现了 UsersModelInterface
var _ UsersModelInterface = (model.UsersModel)(nil)
