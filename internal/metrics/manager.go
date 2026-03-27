package metrics

import (
	"user/internal/config"

	"github.com/zeromicro/go-zero/core/metric"
)

// 增加 "子系统指标" 时不需要关注此文件
// 增加 "子系统" 时需要:
// 1. 增加子系统名称常量
// 2. 为总指标结构体添加对应子系统字段
// 3. 编写指标构建器函数
// 4. 创建子系统代码文件在其内部定义实例创建方法

const (
	serviceSubsystem string = "service" // service 子系统名称
)

// 总指标结构体，各字段代表子系统总指标定义
type MetricsManager struct {
	Service *Servicemetric // 服务收集指标
}

// service 指标构建器
func newSvcMertricsBuilder(c config.Config) *builder {
	return &builder{
		namespace: c.Name,
		subsystem: serviceSubsystem,
	}
}

// 总指标收集器
func NewMetricsManager(c config.Config) *MetricsManager {
	return &MetricsManager{
		Service: newServicemetric(c),
	}
}

// 通用构建器
type builder struct {
	namespace string
	subsystem string
}

// counter vec 创建方法
func (b *builder) newCounterVec(name, help string, labels []string) metric.CounterVec {
	return metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: b.namespace,
		Subsystem: b.subsystem,
		Name:      name,
		Help:      help,
		Labels:    labels, // 定义维度: 可以有多个维度，每个维度组合都是独立的计数器
	})
}

// gauge vec 创建方法
func (b *builder) newGaugeVec(name, help string, labels []string) metric.GaugeVec {
	return metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: b.namespace,
		Subsystem: b.subsystem,
		Name:      name,
		Help:      help,
		Labels:    labels,
	})
}

// histogram vec 创建方法
func (b *builder) newHistogramVec(name, help string, buckets []float64, labels []string) metric.HistogramVec {
	return metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: b.namespace,
		Subsystem: b.subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
		Labels:    labels,
	})
}
