package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword_Success(t *testing.T) {
	// 符合所有规则的密码
	password := "Password123!"
	err := ValidatePassword(password)
	assert.NoError(t, err)
}

func TestValidatePassword_TooShort(t *testing.T) {
	// 少于8位
	password := "Pass1!"
	err := ValidatePassword(password)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPasswordTooShort))
}

func TestValidatePassword_NoUpper(t *testing.T) {
	// 缺少大写字母
	password := "password123!"
	err := ValidatePassword(password)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPasswordNoUpper))
}

func TestValidatePassword_NoLower(t *testing.T) {
	// 缺少小写字母
	password := "PASSWORD123!"
	err := ValidatePassword(password)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPasswordNoLower))
}

func TestValidatePassword_NoDigit(t *testing.T) {
	// 缺少数字
	password := "Password!!!"
	err := ValidatePassword(password)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPasswordNoDigit))
}

func TestValidatePassword_NoSpecial(t *testing.T) {
	// 缺少特殊字符
	password := "Password123"
	err := ValidatePassword(password)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPasswordNoSpecial))
}

func TestValidatePassword_InvalidChar(t *testing.T) {
	// 包含无效字符（空格）
	password := "Password 123!"
	err := ValidatePassword(password)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPasswordInvalidChar))
}

func TestValidatePassword_MultipleErrors(t *testing.T) {
	// 多个错误：太短且缺少多种字符
	password := "abc"
	result := ValidatePasswordDetailed(password)
	assert.False(t, result.Valid)
	assert.GreaterOrEqual(t, len(result.Errors), 1)
}

func TestValidatePasswordDetailed_Success(t *testing.T) {
	password := "Password123!"
	result := ValidatePasswordDetailed(password)

	assert.True(t, result.Valid)
	assert.Equal(t, 12, result.Length)
	assert.True(t, result.HasUpper)
	assert.True(t, result.HasLower)
	assert.True(t, result.HasDigit)
	assert.True(t, result.HasSpecial)
	assert.Empty(t, result.Errors)
}

func TestValidatePasswordDetailed_Failure(t *testing.T) {
	password := "pass"
	result := ValidatePasswordDetailed(password)

	assert.False(t, result.Valid)
	assert.Equal(t, 4, result.Length)
	assert.False(t, result.HasUpper)
	assert.True(t, result.HasLower)
	assert.False(t, result.HasDigit)
	assert.False(t, result.HasSpecial)
	assert.NotEmpty(t, result.Errors)
}

func TestIsStrongPassword_True(t *testing.T) {
	password := "StrongPass123!"
	assert.True(t, IsStrongPassword(password))
}

func TestIsStrongPassword_False(t *testing.T) {
	password := "weak"
	assert.False(t, IsStrongPassword(password))
}

func TestValidatePassword_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "刚好8位且符合规则",
			password: "Pass1!aa",
			wantErr:  false,
		},
		{
			name:     "包含所有允许的特殊字符",
			password: "Pass123!@#$%^&*()_+=[]{};':\"\\|,.<>/?",
			wantErr:  false,
		},
		{
			name:     "中文密码",
			password: "密码Password123!",
			wantErr:  true,
		},
		{
			name:     "emoji密码",
			password: "Password123!😀",
			wantErr:  true,
		},
		{
			name:     "空密码",
			password: "",
			wantErr:  true,
		},
		{
			name:     "只有特殊字符",
			password: "!@#$%^&*",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
