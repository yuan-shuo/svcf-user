package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("成功加密密码", func(t *testing.T) {
		password := "testpassword123"

		hashed, err := HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hashed)
		assert.NotEqual(t, password, hashed)
	})

	t.Run("生成的哈希长度大于0", func(t *testing.T) {
		password := "short"

		hashed, err := HashPassword(password)

		assert.NoError(t, err)
		assert.Greater(t, len(hashed), 0)
	})

	t.Run("相同密码生成不同哈希", func(t *testing.T) {
		password := "samepassword"

		hashed1, err1 := HashPassword(password)
		hashed2, err2 := HashPassword(password)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, hashed1, hashed2, "相同密码应该生成不同的哈希值（因为使用了随机盐）")
	})

	t.Run("空密码可以加密", func(t *testing.T) {
		password := ""

		hashed, err := HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hashed)
	})

	t.Run("长密码可以加密", func(t *testing.T) {
		// bcrypt 最大支持 72 字节
		password := "thisisaverylongpasswordthatexceedsnormallengthlimits123456789"

		hashed, err := HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hashed)
	})

	t.Run("包含特殊字符的密码可以加密", func(t *testing.T) {
		password := "p@$$w0rd!#$%^&*()_+-=[]{}|;':\",./<>?"

		hashed, err := HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hashed)
	})

	t.Run("包含Unicode字符的密码可以加密", func(t *testing.T) {
		password := "密码123🔐"

		hashed, err := HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hashed)
	})
}

func TestComparePassword(t *testing.T) {
	t.Run("正确的密码验证通过", func(t *testing.T) {
		password := "testpassword123"
		hashed, err := HashPassword(password)
		assert.NoError(t, err)

		err = ComparePassword(hashed, password)

		assert.NoError(t, err)
	})

	t.Run("错误的密码验证失败", func(t *testing.T) {
		password := "testpassword123"
		wrongPassword := "wrongpassword"
		hashed, err := HashPassword(password)
		assert.NoError(t, err)

		err = ComparePassword(hashed, wrongPassword)

		assert.Error(t, err)
		assert.Equal(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})

	t.Run("验证不同哈希的相同密码", func(t *testing.T) {
		password := "samepassword"
		hashed1, _ := HashPassword(password)
		hashed2, _ := HashPassword(password)

		// 虽然哈希值不同，但都应该能验证通过
		err1 := ComparePassword(hashed1, password)
		err2 := ComparePassword(hashed2, password)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})

	t.Run("验证空密码", func(t *testing.T) {
		password := ""
		hashed, err := HashPassword(password)
		assert.NoError(t, err)

		err = ComparePassword(hashed, password)

		assert.NoError(t, err)
	})

	t.Run("使用错误的哈希格式返回错误", func(t *testing.T) {
		invalidHash := "invalidhash"
		password := "testpassword"

		err := ComparePassword(invalidHash, password)

		assert.Error(t, err)
	})

	t.Run("使用空哈希返回错误", func(t *testing.T) {
		password := "testpassword"

		err := ComparePassword("", password)

		assert.Error(t, err)
	})
}

func TestHashPasswordAndCompare(t *testing.T) {
	t.Run("完整的加密和验证流程", func(t *testing.T) {
		password := "mypassword"

		// 加密
		hashed, err := HashPassword(password)
		assert.NoError(t, err)

		// 验证正确密码
		err = ComparePassword(hashed, password)
		assert.NoError(t, err)

		// 验证错误密码
		err = ComparePassword(hashed, "wrongpassword")
		assert.Error(t, err)
	})
}
