# svcf-user

microservice frame of user function: 用户微服务框架

[![CI](https://github.com/yuan-shuo/svcf-user/workflows/CI/badge.svg)](https://github.com/yuan-shuo/svcf-user/actions) [![codecov](https://codecov.io/github/yuan-shuo/svcf-user/graph/badge.svg?token=KB7HJSKVPT)](https://codecov.io/github/yuan-shuo/svcf-user)

## 更新计划

1. 其实按理说这个服务已经完成了，但是 "修改密码" 这个接口比较特殊，他需要jwt返回一定信息，但是jwt的解析属于网关的任务，目前接口实现是有很大问题的，所以约等于无效（这个接口目前没法用），先去写网关回来看看如何调整吧...
2. ds说网关要user这边开个JWKS端点，然后用什么非对称加密密钥还是啥让网关能够解析jwt然后给下游？

## 核心模块

1. (svc) user.go 用户系统微服务
2. (svc) cmd/mqs 微服务消费者
3. (job) cmd/migrate 数据库迁移工具

# 其他

如果想部署，直接看ci.yml里的测试流程也可

## 数据库

### 迁移

```bash
go run ./cmd/migrate
```

## 生成

### 数据库模型代码

```bash
goctl model pg datasource -url="postgres://username:123456@127.0.0.1:5432/user_db?sslmode=disable" -table="users" -dir="./internal/model" -cache
```

### 日志、指标代码

```bash
# 工具安装
go install github.com/yuan-shuo/zerotele@latest
# 日志字段代码生成
zerotele lf zerotele.yaml -d ./internal/logger -m mask.go
# 指标管理器代码生成
zerotele met zerotele.yaml -d ./internal/metrics
```

## 牢骚

jwt 退出登录直接前端自己删 localstorage（好像要用cookie的什么httpOnly？从网上听的不知道要不要改动前端），别搞 redis 黑名单没用还复古 session 加 http 网关
