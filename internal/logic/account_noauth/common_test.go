package account_noauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommonBuildBaseKey(t *testing.T) {
	tests := []struct {
		name     string
		codeType string
		want     string
	}{
		{
			name:     "注册类型",
			codeType: "register",
			want:     "account:register",
		},
		{
			name:     "重置密码类型",
			codeType: "reset_password",
			want:     "account:reset_password",
		},
		{
			name:     "提醒已注册类型",
			codeType: "remind_registered",
			want:     "account:remind_registered",
		},
		{
			name:     "空类型",
			codeType: "",
			want:     "account:",
		},
		{
			name:     "包含特殊字符的类型",
			codeType: "type-with_special.chars",
			want:     "account:type-with_special.chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBaseKey(tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommonBuildVerifyKey(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		codeType string
		want     string
	}{
		{
			name:     "注册验证码key",
			email:    "test@example.com",
			codeType: "register",
			want:     "account:register:verify:test@example.com",
		},
		{
			name:     "重置密码验证码key",
			email:    "user@example.com",
			codeType: "reset_password",
			want:     "account:reset_password:verify:user@example.com",
		},
		{
			name:     "包含加号的邮箱",
			email:    "user+tag@example.com",
			codeType: "register",
			want:     "account:register:verify:user+tag@example.com",
		},
		{
			name:     "包含点的邮箱",
			email:    "first.last@example.com",
			codeType: "register",
			want:     "account:register:verify:first.last@example.com",
		},
		{
			name:     "空邮箱",
			email:    "",
			codeType: "register",
			want:     "account:register:verify:",
		},
		{
			name:     "空类型",
			email:    "test@example.com",
			codeType: "",
			want:     "account::verify:test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVerifyKey(tt.email, tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommonBuildLimitKey(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		codeType string
		want     string
	}{
		{
			name:     "注册限流key",
			email:    "test@example.com",
			codeType: "register",
			want:     "account:register:limit:test@example.com",
		},
		{
			name:     "重置密码限流key",
			email:    "user@example.com",
			codeType: "reset_password",
			want:     "account:reset_password:limit:user@example.com",
		},
		{
			name:     "包含加号的邮箱",
			email:    "user+tag@example.com",
			codeType: "register",
			want:     "account:register:limit:user+tag@example.com",
		},
		{
			name:     "包含点的邮箱",
			email:    "first.last@example.com",
			codeType: "register",
			want:     "account:register:limit:first.last@example.com",
		},
		{
			name:     "空邮箱",
			email:    "",
			codeType: "register",
			want:     "account:register:limit:",
		},
		{
			name:     "空类型",
			email:    "test@example.com",
			codeType: "",
			want:     "account::limit:test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLimitKey(tt.email, tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisKeyConstants(t *testing.T) {
	t.Run("验证常量值", func(t *testing.T) {
		assert.Equal(t, "verify", verifyKey)
		assert.Equal(t, "limit", limitKey)
		assert.Equal(t, "code", redisValueCodeFieldName)
		assert.Equal(t, "used", redisValueUsedFieldName)
		assert.Equal(t, "account", redisKeyPrefix)
	})
}

func TestKeyConsistency(t *testing.T) {
	t.Run("验证key构建的一致性", func(t *testing.T) {
		email := "test@example.com"
		codeType := "register"

		// 验证基础key被正确使用
		baseKey := buildBaseKey(codeType)
		verifyKey := buildVerifyKey(email, codeType)
		limitKey := buildLimitKey(email, codeType)

		// verifyKey 应该以 baseKey 开头
		assert.Contains(t, verifyKey, baseKey)
		// limitKey 应该以 baseKey 开头
		assert.Contains(t, limitKey, baseKey)

		// 验证结构一致性
		assert.Equal(t, baseKey+":"+verifyKeyConst+":"+email, verifyKey)
		assert.Equal(t, baseKey+":"+limitKeyConst+":"+email, limitKey)
	})
}

// 在测试中使用局部常量，避免与包级常量冲突
const (
	verifyKeyConst = "verify"
	limitKeyConst  = "limit"
)
