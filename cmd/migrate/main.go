package main

// 需要提前自行创建数据并确保用户权限给够

// 1. 读取配置文件
// 2. 执行数据库迁移

import (
	"flag"
	"os"
	"os/exec"

	"user/internal/config"

	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/user-api.yaml", "配置文件路径")
)

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 一行命令：直接调用 migrate
	cmd := exec.Command("migrate",
		"-source", "file://migrations",
		"-database", c.PostgreSQL.Datasource,
		"up")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
