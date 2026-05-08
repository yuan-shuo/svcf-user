package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// 测试使用的 cost 值，使用最小值加快测试速度
const testCost = 4

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		cost     int
		wantErr  bool
	}{
		{"正常密码加密", "testpassword123", testCost, false},
		{"短密码加密", "short", testCost, false},
		{"空密码加密", "", testCost, false},
		{"长密码加密", "thisisaverylongpasswordthatexceedsnormallengthlimits123456789", testCost, false},
		{"包含特殊字符的密码", "p@$$w0rd!#$%^&*()_+-=[]{}|;':\",./<>?", testCost, false},
		{"包含Unicode字符的密码", "密码123🔐", testCost, false},
		{"cost为MinCost", "test", bcrypt.MinCost, false},
		{"cost为6（边界内）", "test", 6, false},
		{"cost超出上边界", "test", 32, true},
		{"cost为负数", "test", -1, true},
		{"cost为0", "test", 0, true},
		{"cost为3（小于MinCost）", "test", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := HashPassword(tt.password, tt.cost)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hashed)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, hashed)
			assert.NotEqual(t, tt.password, hashed)
			// 验证生成的哈希是有效的 bcrypt 格式
			assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hashed), []byte(tt.password)))
		})
	}
}

func TestHashPassword_SamePasswordDifferentHash(t *testing.T) {
	password := "samepassword"

	hashed1, err1 := HashPassword(password, testCost)
	hashed2, err2 := HashPassword(password, testCost)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, hashed1, hashed2, "相同密码应该生成不同的哈希值（因为使用了随机盐）")
}

func TestComparePassword(t *testing.T) {
	password := "testpassword123"
	hashed, _ := HashPassword(password, testCost)

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		wantErr        bool
		expectedErr    error
	}{
		{"正确的密码验证通过", hashed, password, false, nil},
		{"错误的密码验证失败", hashed, "wrongpassword", true, bcrypt.ErrMismatchedHashAndPassword},
		{"空密码验证失败", hashed, "", true, bcrypt.ErrMismatchedHashAndPassword},
		{"无效哈希格式", "invalidhash", password, true, nil},
		{"空哈希", "", password, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ComparePassword(tt.hashedPassword, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComparePassword_DifferentHashesSamePassword(t *testing.T) {
	password := "samepassword"
	hashed1, _ := HashPassword(password, testCost)
	hashed2, _ := HashPassword(password, testCost)

	// 虽然哈希值不同，但都应该能验证通过
	assert.NoError(t, ComparePassword(hashed1, password))
	assert.NoError(t, ComparePassword(hashed2, password))
}

func TestCheckCost(t *testing.T) {
	tests := []struct {
		name    string
		cost    int
		wantErr bool
	}{
		{"MinCost-1（超出下边界）", bcrypt.MinCost - 1, true},
		{"MinCost（边界值）", bcrypt.MinCost, false},
		{"MinCost+1（边界内）", bcrypt.MinCost + 1, false},
		{"正常值10", 10, false},
		{"6（边界内）", 6, false},
		{"MaxCost-1（边界内）", bcrypt.MaxCost - 1, false},
		{"MaxCost（边界值）", bcrypt.MaxCost, false},
		{"MaxCost+1（超出上边界）", bcrypt.MaxCost + 1, true},
		{"负数", -1, true},
		{"0", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCost(tt.cost)
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, bcrypt.InvalidCostError(0), err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHashPasswordAndCompare_RoundTrip(t *testing.T) {
	passwords := []string{
		"simple",
		"ComplexP@ssw0rd!",
		"1234567890",
		"!@#$%^&*()",
		"中文字符测试",
		"   spaces   ",
		"mixed123!@#中文",
		"",
	}

	for _, pwd := range passwords {
		hashed, err := HashPassword(pwd, testCost)
		assert.NoError(t, err, "密码 '%s' 加密失败", pwd)

		// 验证正确密码
		assert.NoError(t, ComparePassword(hashed, pwd), "密码 '%s' 应该验证通过", pwd)

		// 验证错误密码
		assert.Error(t, ComparePassword(hashed, pwd+"wrong"), "错误密码应该验证失败")
	}
}
