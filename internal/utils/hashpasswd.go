package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 对密码进行加密
// 测试时需要使用小 cost 值, 上边界测试只在 ensureCost 里测，这里不测试边界，cost 尽可能小就行
func HashPassword(password string, cost int) (string, error) {
	// 检查 cost 是否在有效范围内(因为bcrypt在下界不会返回err，宽松且模糊，所以自行构建check)
	err := checkCost(cost)
	if err != nil {
		return "", err
	}
	// bcrypt.GenerateFromPassword源码自带cost边界检查，不需要额外编写工具函数
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// checkCost 检查 cost 是否在有效范围内
func checkCost(cost int) error {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return bcrypt.InvalidCostError(cost)
	}
	return nil
}

// ComparePassword 验证密码是否与哈希匹配, 一致则err=nil
// hashedPassword: 数据库存储的 bcrypt 哈希密码
// password: 用户直接输入的明文密码
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
