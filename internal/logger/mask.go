package logger

import (
	"strconv"
	"strings"
)

// maskUserId 对用户ID（雪花ID）进行脱敏
// 脱敏规则：显示前2位和后2位，中间用 **** 替换
func maskUserId(userid int64) any {
	s := strconv.FormatInt(userid, 10)
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// maskEmail 对用户邮箱进行脱敏
// 脱敏规则：显示邮箱前2个字符和@后的域名，中间用 **** 替换
// 示例：test@example.com -> te****@example.com
func maskEmail(email string) any {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "****"
	}
	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return local + "****@" + domain
	}
	return local[:2] + "****@" + domain
}

// maskUid 对用户雪花ID（JWT中提取）进行脱敏
// 脱敏规则：与 maskUserId 相同，显示前2位和后2位
func maskUid(uid int64) any {
	return maskUserId(uid)
}

// maskRedisKey 对Redis key（可能包含敏感信息）进行脱敏
// 脱敏规则：显示前缀，邮箱部分脱敏
// 示例：user:verify:test@example.com -> user:verify:te****@example.com
func maskRedisKey(rediskey string) any {
	if rediskey == "" {
		return ""
	}
	// 如果包含邮箱，对邮箱部分脱敏
	if strings.Contains(rediskey, "@") {
		parts := strings.Split(rediskey, ":")
		for i, part := range parts {
			if strings.Contains(part, "@") {
				parts[i] = maskEmail(part).(string)
			}
		}
		return strings.Join(parts, ":")
	}
	// 不包含邮箱，显示前10个字符，后面用 **** 替换
	if len(rediskey) <= 10 {
		return rediskey
	}
	return rediskey[:10] + "****"
}
