package mock

import (
	"sync"

	"user/internal/metrics"
)

// testMetrics 用于测试的全局 metrics 实例（避免重复注册）
var testMetrics *metrics.Metrics
var testMetricsOnce sync.Once

// GetTestMetrics 获取单例的 test metrics 实例
func GetTestMetrics() *metrics.Metrics {
	testMetricsOnce.Do(func() {
		testMetrics = metrics.NewMetrics()
	})
	return testMetrics
}
