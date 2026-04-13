package mock

import (
	"sync"

	"user/internal/metrics"
)

// testMetrics 用于测试的全局 metrics 实例（避免重复注册）
var testMetrics *metrics.MetricsManager
var testMetricsOnce sync.Once

// GetTestMetrics 获取单例的 test metrics 实例
func GetTestMetrics() *metrics.MetricsManager {
	testMetricsOnce.Do(func() {
		testMetrics = metrics.NewMetricsManager()
	})
	return testMetrics
}
