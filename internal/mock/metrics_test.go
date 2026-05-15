package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTestMetrics(t *testing.T) {
	t.Run("首次调用返回非nil实例", func(t *testing.T) {
		m := GetTestMetrics()
		assert.NotNil(t, m)
	})

	t.Run("多次调用返回同一实例", func(t *testing.T) {
		metrics1 := GetTestMetrics()
		metrics2 := GetTestMetrics()
		assert.Equal(t, metrics1, metrics2)
		assert.Same(t, metrics1, metrics2)
	})

	t.Run("返回的Metrics包含所有计数器", func(t *testing.T) {
		m := GetTestMetrics()
		assert.NotNil(t, m.UserRegistrationsTotal)
		assert.NotNil(t, m.UserLoginsTotal)
		assert.NotNil(t, m.PasswordChangesTotal)
		assert.NotNil(t, m.VerifyCodeSendsTotal)
		assert.NotNil(t, m.VerifyCodeVerificationsTotal)
		assert.NotNil(t, m.VerifyCodeRateLimitHitsTotal)
		assert.NotNil(t, m.TokenOperationsTotal)
		assert.NotNil(t, m.TokenRefreshTotal)
		assert.NotNil(t, m.DbOperationsTotal)
		assert.NotNil(t, m.BusinessErrorsTotal)
		assert.NotNil(t, m.MqOperationsTotal)
		assert.NotNil(t, m.RequestDurationMs)
		assert.NotNil(t, m.ActiveDbConnections)
		assert.NotNil(t, m.RateLimitHitsTotal)
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
