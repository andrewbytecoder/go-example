package exporter

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type exampleCollector struct {
	subsystem     string                      // 定义子主题， 生成 namespace_subsystem_ 类型的指标名称
	metricDescs   map[string]*prometheus.Desc // 可能会有多个指标，这里用于缓存指标描述，避免重复创建
	metricDescsMu sync.Mutex
	logger        *zap.Logger
}

func init() {
	registerCollector("netclass", NewNetClassCollector)
}

// NewNetClassCollector returns a new Collector exposing network class stats.
func NewNetClassCollector(logger *zap.Logger) (Collector, error) {
	return &exampleCollector{
		subsystem:   "network",
		metricDescs: map[string]*prometheus.Desc{},
		logger:      logger,
	}, nil
}

func (c *exampleCollector) Update(ch chan<- prometheus.Metric) error {
	return c.netClassSysfsUpdate(ch)
}

func (c *exampleCollector) netClassSysfsUpdate(ch chan<- prometheus.Metric) error {
	upDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, c.subsystem, "up"),
		"Value is 1 if operstate is 'up', 0 otherwise.",
		[]string{"device"},
		nil,
	)

	upValue := 0.0

	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, upValue, "eth0")

	pushMetric(ch, c.getFieldDesc("address_assign_type"), "address_assign_type", "type", prometheus.GaugeValue, "eth0")

	return nil
}

func (c *exampleCollector) getFieldDesc(name string) *prometheus.Desc {
	c.metricDescsMu.Lock()
	defer c.metricDescsMu.Unlock()

	fieldDesc, exists := c.metricDescs[name]

	if !exists {
		fieldDesc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, c.subsystem, name),
			fmt.Sprintf("Network device property: %s", name),
			[]string{"device"},
			nil,
		)
		c.metricDescs[name] = fieldDesc
	}

	return fieldDesc
}
