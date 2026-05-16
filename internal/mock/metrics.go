package mock

import (
	"sync"
	"time"

	"user/internal/metrics"
	"user/internal/utils"
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

// NewTestKeyManager 创建用于测试的 RSA KeyManager
func NewTestKeyManager() (*utils.RSAKeyManager, error) {
	cfg := &utils.KeyManagerConfig{
		Bits:             2048,
		RotationInterval: 24 * time.Hour,
		GracePeriod:      48 * time.Hour,
		TokenExpireSecs:  3600,
	}
	return utils.NewRSAKeyManagerWithConfig(cfg)
}
