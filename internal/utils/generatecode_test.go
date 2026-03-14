package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCodeWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    CodeConfig
		wantLen   int
		wantMatch string // 正则匹配模式
		wantErr   bool
	}{
		{
			name: "纯数字验证码-默认6位",
			config: CodeConfig{
				Length:     6,
				UseNumbers: true,
			},
			wantLen:   6,
			wantMatch: `^\d{6}$`,
		},
		{
			name: "纯数字验证码-4位",
			config: CodeConfig{
				Length:     4,
				UseNumbers: true,
			},
			wantLen:   4,
			wantMatch: `^\d{4}$`,
		},
		{
			name: "纯字母大写-8位",
			config: CodeConfig{
				Length:     8,
				UseLetters: true,
				UpperCase:  true,
			},
			wantLen:   8,
			wantMatch: `^[A-Z]{8}$`,
		},
		{
			name: "纯字母小写-8位",
			config: CodeConfig{
				Length:     8,
				UseLetters: true,
				UpperCase:  false,
			},
			wantLen:   8,
			wantMatch: `^[a-z]{8}$`,
		},
		{
			name: "数字+字母混合-大写",
			config: CodeConfig{
				Length:     10,
				UseNumbers: true,
				UseLetters: true,
				UpperCase:  true,
			},
			wantLen:   10,
			wantMatch: `^[0-9A-Z]{10}$`,
		},
		{
			name: "数字+字母混合-小写",
			config: CodeConfig{
				Length:     10,
				UseNumbers: true,
				UseLetters: true,
				UpperCase:  false,
			},
			wantLen:   10,
			wantMatch: `^[0-9a-z]{10}$`,
		},
		{
			name: "排除易混淆字符-纯数字",
			config: CodeConfig{
				Length:     20, // 大量生成以验证排除逻辑
				UseNumbers: true,
				Exclude:    "01",
			},
			wantLen:   20,
			wantMatch: `^[2-9]{20}$`, // 不应包含 0 和 1
		},
		{
			name: "排除易混淆字符-混合模式",
			config: CodeConfig{
				Length:     100, // 大量生成确保排除生效
				UseNumbers: true,
				UseLetters: true,
				UpperCase:  true,
				Exclude:    "0O1Il", // 常见易混淆字符
			},
			wantLen:   100,
			wantMatch: `^[0-9A-Za-z]{100}$`, // 先验证格式
		},
		{
			name: "空配置-默认纯数字6位",
			config: CodeConfig{
				Length: 0, // 会被强制设为6
			},
			wantLen:   6,
			wantMatch: `^\d{6}$`,
		},
		{
			name: "超长验证码",
			config: CodeConfig{
				Length:     100,
				UseNumbers: true,
			},
			wantLen:   100,
			wantMatch: `^\d{100}$`,
		},
		{
			name: "长度为1",
			config: CodeConfig{
				Length:     1,
				UseNumbers: true,
			},
			wantLen:   1,
			wantMatch: `^\d$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateCodeWithConfig(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLen, len(got), "验证码长度不匹配")
			assert.Regexp(t, tt.wantMatch, got, "验证码格式不匹配")

			// 特殊检查：验证排除字符确实被排除了
			if tt.config.Exclude != "" {
				for _, excludeChar := range tt.config.Exclude {
					assert.NotContains(t, got, string(excludeChar),
						"验证码不应包含被排除的字符: %c", excludeChar)
				}
			}
		})
	}
}

func TestGenerateDigitCode(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"6位数字", 6},
		{"4位数字", 4},
		{"10位数字", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateDigitCode(tt.length)

			assert.Equal(t, tt.length, len(got))
			assert.Regexp(t, `^\d+$`, got)

			// 验证纯数字
			_, err := regexp.MatchString(`^\d+$`, got)
			require.NoError(t, err)
		})
	}
}

func TestGenerateMixedCode(t *testing.T) {
	// 生成大量样本验证排除逻辑
	sampleSize := 100
	codes := make([]string, sampleSize)

	for i := 0; i < sampleSize; i++ {
		codes[i] = GenerateMixedCode(20)
	}

	// 验证基本格式
	for _, code := range codes {
		assert.Equal(t, 20, len(code))
		assert.Regexp(t, `^[0-9A-Z]+$`, code) // 大写字母+数字

		// 验证不包含排除字符
		assert.NotContains(t, code, "0")
		assert.NotContains(t, code, "O")
		assert.NotContains(t, code, "1")
		assert.NotContains(t, code, "I")
	}

	// 验证随机性：统计不同字符的出现
	charCount := make(map[rune]int)
	for _, code := range codes {
		for _, c := range code {
			charCount[c]++
		}
	}

	// 应该有多个不同的字符被使用（不是每次都生成一样的）
	assert.Greater(t, len(charCount), 10, "验证码应使用多种不同字符")

	// 验证同时包含数字和字母
	hasDigit := false
	hasLetter := false
	for char := range charCount {
		if char >= '0' && char <= '9' {
			hasDigit = true
		}
		if char >= 'A' && char <= 'Z' {
			hasLetter = true
		}
	}
	assert.True(t, hasDigit, "应包含数字")
	assert.True(t, hasLetter, "应包含字母")
}

func TestGenerateCodeRandomness(t *testing.T) {
	// 验证随机性：连续生成不应相同
	config := CodeConfig{
		Length:     10,
		UseNumbers: true,
	}

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := GenerateCodeWithConfig(config)
		require.NoError(t, err)

		// 极大概率不会重复（虽然理论上可能，但概率极低）
		if seen[code] && i > 0 {
			t.Logf("第 %d 次生成出现重复: %s", i, code)
		}
		seen[code] = true
	}

	// 100个10位数字码，重复率应极低，实际唯一值应接近100
	uniqueRatio := float64(len(seen)) / 100.0
	assert.Greater(t, uniqueRatio, 0.95, "随机性不足，重复率过高")
}

func TestGenerateCodeWithConfig_EdgeCases(t *testing.T) {
	t.Run("排除所有数字后只剩字母", func(t *testing.T) {
		config := CodeConfig{
			Length:     10,
			UseNumbers: true,
			UseLetters: true,
			Exclude:    "0123456789",
		}
		got, err := GenerateCodeWithConfig(config)
		require.NoError(t, err)
		assert.Regexp(t, `^[a-z]+$`, got) // 应该只有小写字母
	})

	t.Run("排除字符大小写敏感", func(t *testing.T) {
		config := CodeConfig{
			Length:     50,
			UseLetters: true,
			UpperCase:  true,
			Exclude:    "A", // 只排除大写A
		}
		got, err := GenerateCodeWithConfig(config)
		require.NoError(t, err)
		assert.NotContains(t, got, "A")
		// 但应包含其他大写字母
		hasOtherUpper := false
		for _, c := range got {
			if c >= 'B' && c <= 'Z' {
				hasOtherUpper = true
				break
			}
		}
		assert.True(t, hasOtherUpper)
	})
}

// Benchmark 测试
func BenchmarkGenerateDigitCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateDigitCode(6)
	}
}

func BenchmarkGenerateMixedCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateMixedCode(6)
	}
}

func BenchmarkGenerateCodeWithConfig(b *testing.B) {
	config := CodeConfig{
		Length:     6,
		UseNumbers: true,
		UseLetters: true,
		UpperCase:  true,
		Exclude:    "0O1I",
	}
	for i := 0; i < b.N; i++ {
		GenerateCodeWithConfig(config)
	}
}
