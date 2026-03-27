package metrics

import (
	"user/internal/config"

	"github.com/zeromicro/go-zero/core/metric"
)

type Servicemetric struct {
	Example metric.CounterVec // 示例指标
	// Example Example // 示例指标
}

// type Example struct {
// 	metric.CounterVec
// }

// func (m Example) Inc(example_label1 string) {
// 	m.CounterVec.Inc(example_label1)
// }

// 服务指标构建器
func newServicemetric(c config.Config) *Servicemetric {
	b := newSvcMertricsBuilder(c)
	return &Servicemetric{
		Example: b.newCounterVec(
			"example_name",
			"example_help",
			[]string{"example_label1"},
		),
		// Example: Example{b.newCounterVec(
		// 	"example_name",
		// 	"example_help",
		// 	[]string{"example_label1"},
		// )},
	}
}
