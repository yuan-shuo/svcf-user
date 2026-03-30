# svcf-user
microservice frame of user function: 用户微服务框架

[![CI](https://github.com/yuan-shuo/svcf-user/workflows/CI/badge.svg)](https://github.com/yuan-shuo/svcf-user/actions) [![codecov](https://codecov.io/github/yuan-shuo/svcf-user/graph/badge.svg?token=KB7HJSKVPT)](https://codecov.io/github/yuan-shuo/svcf-user)

## 更新计划

1. 限流
1. 可能存在的优化
1. jwt通用组件

## 核心模块

1. (svc) user.go 用户系统微服务 
2. (svc) cmd/mqs 微服务消费者
3. (job) cmd/migrate 数据库迁移工具 (需要配合go-migrate工具使用)

# 其他

如果想部署，直接看ci.yml里的测试流程也可

## 数据库

### 迁移

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go run ./cmd/migrate
```

## 生成

### 数据库模型代码

```bash
goctl model pg datasource -url="postgres://username:123456@127.0.0.1:5432/user_db?sslmode=disable" -table="users" -dir="./internal/model" -cache
```

### prom指标代码

```bash
go install github.com/yuan-shuo/gometrics@latest
gometrics -f metrics.yaml -d ./internal/metrics
```

