package utils

import (
	"errors"
	"regexp"
	"unicode"
)

var (
	// ErrPasswordTooShort 密码太短
	ErrPasswordTooShort = errors.New("password too short")
	// ErrPasswordNoUpper 缺少大写字母
	ErrPasswordNoUpper = errors.New("password must contain uppercase letter")
	// ErrPasswordNoLower 缺少小写字母
	ErrPasswordNoLower = errors.New("password must contain lowercase letter")
	// ErrPasswordNoDigit 缺少数字
	ErrPasswordNoDigit = errors.New("password must contain digit")
	// ErrPasswordNoSpecial 缺少特殊字符
	ErrPasswordNoSpecial = errors.New("password must contain special character")
	// ErrPasswordInvalidChar 包含无效字符
	ErrPasswordInvalidChar = errors.New("password contains invalid character")

	// specialChars 特殊字符正则
	specialChars = regexp.MustCompile(`[!@#$%^&*()_+=\[\]{};':"\\|,.<>/?]`)
	// validChars 有效字符正则（字母、数字、特殊字符）
	validChars = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+=\[\]{};':"\\|,.<>/?]+$`)
)

// PasswordValidationResult 密码校验结果
type PasswordValidationResult struct {
	Valid      bool     // 是否通过校验
	Errors     []error  // 所有错误
	Length     int      // 密码长度
	HasUpper   bool     // 是否有大写字母
	HasLower   bool     // 是否有小写字母
	HasDigit   bool     // 是否有数字
	HasSpecial bool     // 是否有特殊字符
}

// ValidatePassword 校验密码强度
// 规则：至少8位，包含大小写字母、数字和特殊字符
// 返回普通 error，不包含业务错误码
func ValidatePassword(password string) error {
	result := ValidatePasswordDetailed(password)
	if result.Valid {
		return nil
	}
	if len(result.Errors) > 0 {
		return result.Errors[0]
	}
	return errors.New("password validation failed")
}

// ValidatePasswordDetailed 详细密码校验，返回完整结果
// 可用于前端展示密码强度或具体错误提示
func ValidatePasswordDetailed(password string) PasswordValidationResult {
	result := PasswordValidationResult{
		Length: len(password),
		Errors: make([]error, 0),
	}

	// 检查长度
	if result.Length < 8 {
		result.Errors = append(result.Errors, ErrPasswordTooShort)
	}

	// 检查字符有效性并统计
	if !validChars.MatchString(password) {
		result.Errors = append(result.Errors, ErrPasswordInvalidChar)
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			result.HasUpper = true
		case unicode.IsLower(char):
			result.HasLower = true
		case unicode.IsDigit(char):
			result.HasDigit = true
		}
	}

	if specialChars.MatchString(password) {
		result.HasSpecial = true
	}

	// 检查各类字符是否齐全
	if !result.HasUpper {
		result.Errors = append(result.Errors, ErrPasswordNoUpper)
	}
	if !result.HasLower {
		result.Errors = append(result.Errors, ErrPasswordNoLower)
	}
	if !result.HasDigit {
		result.Errors = append(result.Errors, ErrPasswordNoDigit)
	}
	if !result.HasSpecial {
		result.Errors = append(result.Errors, ErrPasswordNoSpecial)
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// IsStrongPassword 快速判断密码是否足够强
func IsStrongPassword(password string) bool {
	return ValidatePassword(password) == nil
}
