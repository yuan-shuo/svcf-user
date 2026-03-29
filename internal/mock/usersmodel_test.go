package mock

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"user/internal/model"

	"github.com/stretchr/testify/assert"
)

func TestUsersModel_Insert(t *testing.T) {
	t.Run("成功插入", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		user := &model.Users{
			Id:           1,
			SnowflakeId:  123456789,
			Nickname:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
		}

		result := &SqlResult{LastID: 1, RA: 1}
		m.On("Insert", ctx, user).Return(result, nil)

		res, err := m.Insert(ctx, user)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, int64(1), res.(*SqlResult).LastID)
		m.AssertExpectations(t)
	})

	t.Run("插入失败", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		user := &model.Users{
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
		}

		m.On("Insert", ctx, user).Return(nil, errors.New("insert failed"))

		res, err := m.Insert(ctx, user)

		assert.Error(t, err)
		assert.Nil(t, res)
		m.AssertExpectations(t)
	})

	t.Run("返回 nil result", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		user := &model.Users{}

		m.On("Insert", ctx, user).Return(nil, nil)

		res, err := m.Insert(ctx, user)

		assert.NoError(t, err)
		assert.Nil(t, res)
		m.AssertExpectations(t)
	})
}

func TestUsersModel_FindOne(t *testing.T) {
	t.Run("找到用户", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		id := int64(1)
		expectedUser := &model.Users{
			Id:           id,
			SnowflakeId:  123456789,
			Nickname:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
		}

		m.On("FindOne", ctx, id).Return(expectedUser, nil)

		user, err := m.FindOne(ctx, id)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.Id, user.Id)
		assert.Equal(t, expectedUser.Email, user.Email)
		m.AssertExpectations(t)
	})

	t.Run("用户不存在", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		id := int64(999)

		m.On("FindOne", ctx, id).Return(nil, model.ErrNotFound)

		user, err := m.FindOne(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, model.ErrNotFound, err)
		assert.Nil(t, user)
		m.AssertExpectations(t)
	})

	t.Run("数据库错误", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		id := int64(1)

		m.On("FindOne", ctx, id).Return(nil, errors.New("database error"))

		user, err := m.FindOne(ctx, id)

		assert.Error(t, err)
		assert.Nil(t, user)
		m.AssertExpectations(t)
	})
}

func TestUsersModel_FindOneByEmail(t *testing.T) {
	t.Run("通过邮箱找到用户", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		email := "test@example.com"
		expectedUser := &model.Users{
			Id:           1,
			SnowflakeId:  123456789,
			Nickname:     "testuser",
			Email:        email,
			PasswordHash: "hashedpassword",
		}

		m.On("FindOneByEmail", ctx, email).Return(expectedUser, nil)

		user, err := m.FindOneByEmail(ctx, email)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		m.AssertExpectations(t)
	})

	t.Run("邮箱不存在", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		email := "nonexistent@example.com"

		m.On("FindOneByEmail", ctx, email).Return(nil, model.ErrNotFound)

		user, err := m.FindOneByEmail(ctx, email)

		assert.Error(t, err)
		assert.Nil(t, user)
		m.AssertExpectations(t)
	})
}

func TestUsersModel_FindOneBySnowflakeId(t *testing.T) {
	t.Run("通过雪花ID找到用户", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		snowflakeId := int64(123456789)
		expectedUser := &model.Users{
			Id:           1,
			SnowflakeId:  snowflakeId,
			Nickname:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
		}

		m.On("FindOneBySnowflakeId", ctx, snowflakeId).Return(expectedUser, nil)

		user, err := m.FindOneBySnowflakeId(ctx, snowflakeId)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, snowflakeId, user.SnowflakeId)
		m.AssertExpectations(t)
	})

	t.Run("雪花ID不存在", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		snowflakeId := int64(999999999)

		m.On("FindOneBySnowflakeId", ctx, snowflakeId).Return(nil, model.ErrNotFound)

		user, err := m.FindOneBySnowflakeId(ctx, snowflakeId)

		assert.Error(t, err)
		assert.Nil(t, user)
		m.AssertExpectations(t)
	})
}

func TestUsersModel_Update(t *testing.T) {
	t.Run("成功更新", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		user := &model.Users{
			Id:           1,
			Nickname:     "updateduser",
			Email:        "test@example.com",
			PasswordHash: "newhashedpassword",
		}

		m.On("Update", ctx, user).Return(nil)

		err := m.Update(ctx, user)

		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("更新失败", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		user := &model.Users{
			Id: 1,
		}

		m.On("Update", ctx, user).Return(errors.New("update failed"))

		err := m.Update(ctx, user)

		assert.Error(t, err)
		m.AssertExpectations(t)
	})
}

func TestUsersModel_Delete(t *testing.T) {
	t.Run("成功删除", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		id := int64(1)

		m.On("Delete", ctx, id).Return(nil)

		err := m.Delete(ctx, id)

		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	t.Run("删除失败", func(t *testing.T) {
		m := new(UsersModel)
		ctx := context.Background()
		id := int64(1)

		m.On("Delete", ctx, id).Return(errors.New("delete failed"))

		err := m.Delete(ctx, id)

		assert.Error(t, err)
		m.AssertExpectations(t)
	})
}

// 验证 UsersModel 实现了 model.UsersModel 接口
var _ model.UsersModel = (*UsersModel)(nil)

// 验证 SqlResult 实现了 sql.Result 接口
var _ sql.Result = (*SqlResult)(nil)
