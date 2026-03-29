package mock

import (
	"errors"
	"testing"

	"user/internal/errs"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

// 模拟 testing.T 用于测试
type mockTester struct {
	cleanupFuncs []func()
}

func (m *mockTester) Helper() {}

func (m *mockTester) Cleanup(f func()) {
	m.cleanupFuncs = append(m.cleanupFuncs, f)
}

func (m *mockTester) Fatalf(format string, args ...interface{}) {}

func (m *mockTester) Logf(format string, args ...interface{}) {}

func (m *mockTester) RunCleanup() {
	for _, f := range m.cleanupFuncs {
		f()
	}
}

func TestIsCodeError(t *testing.T) {
	t.Run("nil错误", func(t *testing.T) {
		result := IsCodeError(nil, 1001)
		assert.False(t, result)
	})

	t.Run("是CodeError且匹配", func(t *testing.T) {
		err := errs.New(1001, "test error")
		result := IsCodeError(err, 1001)
		assert.True(t, result)
	})

	t.Run("是CodeError但不匹配", func(t *testing.T) {
		err := errs.New(1002, "test error")
		result := IsCodeError(err, 1001)
		assert.False(t, result)
	})

	t.Run("普通错误", func(t *testing.T) {
		err := errors.New("normal error")
		result := IsCodeError(err, 1001)
		assert.False(t, result)
	})
}

func TestSetupTestRedis(t *testing.T) {
	t.Run("创建Redis实例", func(t *testing.T) {
		tester := &mockTester{}
		r, s := SetupTestRedis(tester)

		assert.NotNil(t, r)
		assert.NotNil(t, s)
		assert.NotEmpty(t, s.Addr())

		// 测试Redis连接
		err := r.Set("test", "value")
		assert.NoError(t, err)

		val, err := r.Get("test")
		assert.NoError(t, err)
		assert.Equal(t, "value", val)

		// 清理
		tester.RunCleanup()
	})

	t.Run("多个Redis实例", func(t *testing.T) {
		tester1 := &mockTester{}
		tester2 := &mockTester{}

		r1, s1 := SetupTestRedis(tester1)
		r2, s2 := SetupTestRedis(tester2)

		assert.NotNil(t, r1)
		assert.NotNil(t, r2)
		// 验证是不同的实例
		assert.NotEqual(t, s1.Addr(), s2.Addr())

		// 分别设置不同的值
		err := r1.Set("key1", "value1")
		assert.NoError(t, err)

		err = r2.Set("key2", "value2")
		assert.NoError(t, err)

		// 验证各自的值
		val1, err := r1.Get("key1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := r2.Get("key2")
		assert.NoError(t, err)
		assert.Equal(t, "value2", val2)

		// 清理
		tester1.RunCleanup()
		tester2.RunCleanup()
	})

	t.Run("Redis操作", func(t *testing.T) {
		tester := &mockTester{}
		r, _ := SetupTestRedis(tester)

		// 测试 Set 和 Get
		err := r.Set("key", "value")
		assert.NoError(t, err)

		val, err := r.Get("key")
		assert.NoError(t, err)
		assert.Equal(t, "value", val)

		// 测试 Del
		r.Del("key")

		// 验证 key 已被删除（获取时可能返回空值而不是错误）
		val, _ = r.Get("key")
		assert.Empty(t, val)

		// 清理
		tester.RunCleanup()
	})
}

// 验证 mockTester 实现了 miniredis.Tester 接口
var _ miniredis.Tester = (*mockTester)(nil)
