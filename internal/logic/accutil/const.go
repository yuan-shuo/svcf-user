package accutil

const (
	VerifyKey               string = "verify"  // 包级验证redis键词缀
	LimitKey                string = "limit"   // 包级限流redis键词缀
	RedisValueCodeFieldName string = "code"    // redis hash 验证码值的键名
	RedisValueUsedFieldName string = "used"    // redis hash 是否使用过值的键名 "0": 未使用过
	RedisKeyPrefix          string = "account" // redis 键名前缀
)
