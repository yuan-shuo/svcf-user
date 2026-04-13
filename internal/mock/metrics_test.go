package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTestMetrics(t *testing.T) {
	t.Run("首次调用返回非nil实例", func(t *testing.T) {
		metrics := GetTestMetrics()
		assert.NotNil(t, metrics)
	})

	t.Run("多次调用返回同一实例", func(t *testing.T) {
		metrics1 := GetTestMetrics()
		metrics2 := GetTestMetrics()
		assert.Equal(t, metrics1, metrics2)
		assert.Same(t, metrics1, metrics2)
	})

	t.Run("返回的MetricsManager包含所有子系统", func(t *testing.T) {
		metrics := GetTestMetrics()
		assert.NotNil(t, metrics.Account)
		assert.NotNil(t, metrics.AccountNoauth)
		assert.NotNil(t, metrics.Verifycode)
	})

	t.Run("Account子系统包含所有计数器", func(t *testing.T) {
		metrics := GetTestMetrics()
		assert.NotNil(t, metrics.Account.PasswordChangesTotal)
	})

	t.Run("AccountNoauth子系统包含所有计数器", func(t *testing.T) {
		metrics := GetTestMetrics()
		assert.NotNil(t, metrics.AccountNoauth.LoginsTotal)
		assert.NotNil(t, metrics.AccountNoauth.RegistrationsTotal)
		assert.NotNil(t, metrics.AccountNoauth.PasswordResetsTotal)
		assert.NotNil(t, metrics.AccountNoauth.TokenRefreshesTotal)
	})

	t.Run("Verifycode子系统包含所有计数器", func(t *testing.T) {
		metrics := GetTestMetrics()
		assert.NotNil(t, metrics.Verifycode.CodesSentTotal)
		assert.NotNil(t, metrics.Verifycode.CodeVerificationsTotal)
		assert.NotNil(t, metrics.Verifycode.RateLimitHitsTotal)
	})
}

func TestGetTestMetrics_Concurrent(t *testing.T) {
	t.Run("并发调用返回同一实例", func(t *testing.T) {
		const numGoroutines = 100
		results := make(chan *struct{}, numGoroutines)
		var firstMetrics = GetTestMetrics()

		for i := 0; i < numGoroutines; i++ {
			go func() {
				m := GetTestMetrics()
				if m == firstMetrics {
					results <- &struct{}{}
				}
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			<-results
		}

		// 如果所有goroutine都返回相同的实例，则测试通过
		assert.Equal(t, numGoroutines, cap(results))
	})
}
