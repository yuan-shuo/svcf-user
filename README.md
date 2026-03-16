# svcf-user
service frame of user: 用户微服务框架

[![CI](https://github.com/yuan-shuo/svcf-user/workflows/CI/badge.svg)](https://github.com/yuan-shuo/svcf-user/actions) [![codecov](https://codecov.io/github/yuan-shuo/svcf-user/graph/badge.svg?token=KB7HJSKVPT)](https://codecov.io/github/yuan-shuo/svcf-user)

## 更新计划

1. 邮箱验证码提前检查邮箱格式及是否注册过
2. 邮箱注册单元测试，其他功能单元测试

## 数据库模型生成

```bash
goctl model pg datasource -url="postgres://username:123456@127.0.0.1:5432/user_db?sslmode=disable" -table="users" -dir="./internal/model" -cache
```

## redis 键命名

```go
// account:register:verify:3695@qq.com - 服务:子功能:功能类型:参数
// account:register:limit:3695@qq.com
baseKey := fmt.Sprintf("%s:%s", l.svcCtx.Config.Register.SendCodeConfig.RedisKeyPrefix, l.svcCtx.Config.Register.SendCodeConfig.ReceiveType)
// redis缓存验证码数据
redisKey := fmt.Sprintf("%s:verify:%s", baseKey, req.Email)
// 设置限流验证码键
limitKey := fmt.Sprintf("%s:limit:%s", baseKey, req.Email)
```