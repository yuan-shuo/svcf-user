package utils

import (
	"crypto/rand"
	"math/big"
)

type CodeConfig struct {
	Length     int    // 验证码长度
	UseLetters bool   // 是否包含字母
	UseNumbers bool   // 是否包含数字
	UpperCase  bool   // 是否大写（如果包含字母）
	Exclude    string // 排除的字符（如 0, O, I 等容易混淆的）
}

func GenerateCodeWithConfig(config CodeConfig) (string, error) {
	// 构建字符集
	charset := ""

	if config.UseNumbers {
		charset += "0123456789"
	}

	if config.UseLetters {
		letters := "abcdefghijklmnopqrstuvwxyz"
		if config.UpperCase {
			letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		}
		charset += letters
	}

	// 默认：纯数字6位
	if charset == "" {
		charset = "0123456789"
		config.Length = 6
	}

	// 排除易混淆字符
	if config.Exclude != "" {
		excludeMap := make(map[rune]bool)
		for _, c := range config.Exclude {
			excludeMap[c] = true
		}

		filtered := make([]rune, 0, len(charset))
		for _, c := range charset {
			if !excludeMap[c] {
				filtered = append(filtered, c)
			}
		}
		charset = string(filtered)
	}

	// 生成验证码
	result := make([]byte, config.Length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < config.Length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}

// 常用的快捷方法
func GenerateDigitCode(length int) string {
	code, _ := GenerateCodeWithConfig(CodeConfig{
		Length:     length,
		UseNumbers: true,
	})
	return code
}

func GenerateMixedCode(length int) string {
	code, _ := GenerateCodeWithConfig(CodeConfig{
		Length:     length,
		UseNumbers: true,
		UseLetters: true,
		UpperCase:  true,
		Exclude:    "0O1I", // 排除容易混淆的字符
	})
	return code
}

// // 使用示例
// func main() {
// 	// 6位数字验证码
// 	code1 := GenerateDigitCode(6)
// 	fmt.Println("数字验证码:", code1) // "482731"

// 	// 6位混合验证码（排除易混淆字符）
// 	code2 := GenerateMixedCode(6)
// 	fmt.Println("混合验证码:", code2) // "A3B7C9"
// }
