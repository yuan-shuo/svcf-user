package db

import (
	"database/sql"
	"time"
	"user/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const (
	// 默认连接池配置
	defaultMaxOpenConns    = 64   // 最大打开连接数，默认64
	defaultMaxIdleConns    = 64   // 最大空闲连接数，默认64
	defaultConnMaxLifetime = 3600 // 连接最大生命周期(秒)，默认1小时
	defaultConnMaxIdleTime = 600  // 空闲连接最大生命周期(秒)，默认10分钟
)

// NewPostgreSQL 创建自定义 PostgreSQL 连接，支持连接池配置
func NewPostgreSQL(conf config.PostgreSQL) sqlx.SqlConn {
	// 打开数据库连接
	db, err := sql.Open("pgx", conf.Datasource)
	if err != nil {
		logx.Errorf("打开 PostgreSQL 连接失败: %v", err)
		panic(err)
	}

	// 配置连接池参数
	configurePool(db, conf.Pool)

	// 验证连接
	if err := db.Ping(); err != nil {
		logx.Errorf("连接 PostgreSQL 失败: %v", err)
		_ = db.Close()
		panic(err)
	}

	logx.Infof("PostgreSQL 连接成功，连接池配置: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%ds, MaxIdleTime=%ds",
		getMaxOpenConns(conf.Pool.MaxOpenConns),
		getMaxIdleConns(conf.Pool.MaxIdleConns),
		getConnMaxLifetime(conf.Pool.ConnMaxLifetime),
		getConnMaxIdleTime(conf.Pool.ConnMaxIdleTime),
	)

	// 使用 NewSqlConnFromDB 创建 sqlx.SqlConn
	return sqlx.NewSqlConnFromDB(db)
}

// configurePool 配置连接池参数
func configurePool(db *sql.DB, pool config.PostgreSQLPool) {
	// 设置最大打开连接数
	maxOpenConns := getMaxOpenConns(pool.MaxOpenConns)
	db.SetMaxOpenConns(maxOpenConns)

	// 设置最大空闲连接数
	maxIdleConns := getMaxIdleConns(pool.MaxIdleConns)
	db.SetMaxIdleConns(maxIdleConns)

	// 设置连接最大生命周期
	connMaxLifetime := getConnMaxLifetime(pool.ConnMaxLifetime)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)

	// 设置连接最大空闲时间 (Go 1.15+)
	connMaxIdleTime := getConnMaxIdleTime(pool.ConnMaxIdleTime)
	db.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Second)
}

// getMaxOpenConns 获取最大打开连接数
func getMaxOpenConns(val int) int {
	if val <= 0 {
		return defaultMaxOpenConns
	}
	return val
}

// getMaxIdleConns 获取最大空闲连接数
func getMaxIdleConns(val int) int {
	if val <= 0 {
		return defaultMaxIdleConns
	}
	return val
}

// getConnMaxLifetime 获取连接最大生命周期
func getConnMaxLifetime(val int) int {
	if val <= 0 {
		return defaultConnMaxLifetime
	}
	return val
}

// getConnMaxIdleTime 获取连接最大空闲时间
func getConnMaxIdleTime(val int) int {
	if val <= 0 {
		return defaultConnMaxIdleTime
	}
	return val
}
