package db

import (
	"user/internal/config"

	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func NewPostgreSQL(conf config.PostgreSQL) sqlx.SqlConn {
	return postgres.New(conf.Datasource)
}
