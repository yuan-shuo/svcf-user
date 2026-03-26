package account_noauth

import "fmt"

const (
	verifyKey               string = "verify"  // 包级验证redis键词缀
	limitKey                string = "limit"   // 包级限流redis键词缀
	redisValueCodeFieldName string = "code"    // redis hash 验证码值的键名
	redisValueUsedFieldName string = "used"    // redis hash 是否使用过值的键名 "0": 未使用过
	redisKeyPrefix          string = "account" // redis 键名前缀
)

func buildBaseKey(codeType string) string {
	return fmt.Sprintf("%s:%s",
		redisKeyPrefix,
		codeType,
	)
}

func buildVerifyKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), verifyKey, email)
}

func buildLimitKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), limitKey, email)
}
