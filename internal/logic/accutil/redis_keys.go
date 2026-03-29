package accutil

import (
	"fmt"
)

// buildBaseKey 构建基础 key
func buildBaseKey(codeType string) string {
	return fmt.Sprintf("%s:%s",
		RedisKeyPrefix,
		codeType,
	)
}

// BuildVerifyKey 构建验证码验证 key
func BuildVerifyKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), VerifyKey, email)
}

// BuildLimitKey 构建限流 key
func BuildLimitKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), LimitKey, email)
}
