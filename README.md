# svcf-user

microservice frame of user function: 用户微服务框架

[![CI](https://github.com/yuan-shuo/svcf-user/workflows/CI/badge.svg)](https://github.com/yuan-shuo/svcf-user/actions) [![codecov](https://codecov.io/github/yuan-shuo/svcf-user/graph/badge.svg?token=KB7HJSKVPT)](https://codecov.io/github/yuan-shuo/svcf-user)

## 更新计划

1. 指标
2. 可能存在的优化

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
