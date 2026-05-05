FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

# 设置时区（可选，默认 UTC）
ENV TZ=Asia/Shanghai

WORKDIR /app

# 复制二进制和配置
COPY bin/user-migrate /app/user-migrate
COPY etc/user-api.template.yaml /app/etc/user-api.yaml
# 复制 migrations 目录下的所有迁移文件
COPY migrations/ /app/migrations/

# 确保二进制可执行
RUN chmod +x /app/user-migrate

CMD ["/app/user-migrate"]