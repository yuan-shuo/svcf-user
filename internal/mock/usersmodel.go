package mock

import (
	"context"
	"database/sql"

	"user/internal/model"

	"github.com/stretchr/testify/mock"
)

// UsersModel 模拟 UsersModel 接口
type UsersModel struct {
	mock.Mock
}

func (m *UsersModel) Insert(ctx context.Context, data *model.Users) (sql.Result, error) {
	args := m.Called(ctx, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *UsersModel) FindOne(ctx context.Context, id int64) (*model.Users, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *UsersModel) FindOneByEmail(ctx context.Context, email string) (*model.Users, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *UsersModel) FindOneBySnowflakeId(ctx context.Context, snowflakeId int64) (*model.Users, error) {
	args := m.Called(ctx, snowflakeId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *UsersModel) Update(ctx context.Context, data *model.Users) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *UsersModel) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
