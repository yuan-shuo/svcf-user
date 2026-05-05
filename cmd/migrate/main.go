package main

import (
	"flag"
	"log"

	"user/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/user-api.yaml", "配置文件路径")
)

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 直接使用 go-migrate 库
	m, err := migrate.New(
		"file://migrations",
		c.PostgreSQL.Datasource,
	)
	if err != nil {
		log.Fatalf("创建 migrate 实例失败: %v", err)
	}
	defer m.Close()

	// 执行迁移
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("迁移失败: %v", err)
	}

	log.Println("迁移成功！")
}
