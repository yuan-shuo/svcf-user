package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskUserId(t *testing.T) {
	tests := []struct {
		name   string
		input  int64
		expect any
	}{
		{"正常雪花ID", 1234567890123456789, "12****89"},
		{"短ID-4位", 1234, "****"},
		{"短ID-3位", 123, "****"},
		{"短ID-1位", 1, "****"},
		{"零值", 0, "****"},
		{"负数", -12345678, "-1****78"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskUserId(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect any
	}{
		{"正常邮箱", "test@example.com", "te****@example.com"},
		{"短用户名-2字符", "ab@example.com", "ab****@example.com"},
		{"短用户名-1字符", "a@example.com", "a****@example.com"},
		{"长用户名", "longusername@example.com", "lo****@example.com"},
		{"空字符串", "", ""},
		{"无@符号", "invalid-email", "****"},
		{"多个@符号", "a@b@c.com", "****"},
		{"带+号的邮箱", "user+tag@example.com", "us****@example.com"},
		{"企业邮箱", "john.doe@company.co.uk", "jo****@company.co.uk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskEmail(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestMaskUid(t *testing.T) {
	tests := []struct {
		name   string
		input  int64
		expect any
	}{
		{"正常UID", 1234567890123456789, "12****89"},
		{"短UID", 1234, "****"},
		{"零值", 0, "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskUid(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestMaskRedisKey(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect any
	}{
		{"包含邮箱的key", "user:verify:test@example.com", "user:verify:te****@example.com"},
		{"包含邮箱的limit key", "user:limit:user@domain.com", "user:limit:us****@domain.com"},
		{"普通短key", "user:123", "user:123"},
		{"普通长key", "user:verify:1234567890abcdef", "user:verif****"},
		{"空字符串", "", ""},
		{"恰好10字符", "1234567890", "1234567890"},
		{"11字符", "12345678901", "1234567890****"},
		{"多个冒号带邮箱", "a:b:c:test@example.com:d", "a:b:c:te****@example.com:d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskRedisKey(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// TestSensitiveInterface 测试生成的类型是否正确实现了 Sensitive 接口
func TestSensitiveInterface(t *testing.T) {
	tests := []struct {
		name     string
		field    interface{ MaskSensitive() any }
		expected any
	}{
		{"UserId", UserId(1234567890123456789), "12****89"},
		{"Email", Email("test@example.com"), "te****@example.com"},
		{"Uid", Uid(1234567890123456789), "12****89"},
		{"RedisKey", RedisKey("user:verify:test@example.com"), "user:verify:te****@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.field.MaskSensitive()
			assert.Equal(t, tt.expected, result)
		})
	}
}
