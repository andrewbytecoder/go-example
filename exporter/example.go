// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter

import (
	"fmt"
	"net"
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type netClassCollector struct {
	subsystem             string
	ignoredDevicesPattern *regexp.Regexp
	metricDescs           map[string]*prometheus.Desc
	metricDescsMu         sync.Mutex
	logger                *zap.Logger
}

func init() {
	registerCollector("netclass", defaultEnabled, NewNetClassCollector)
}

// NewNetClassCollector returns a new Collector exposing network class stats.
func NewNetClassCollector(logger *zap.Logger) (Collector, error) {
	pattern := regexp.MustCompile("^$")
	return &netClassCollector{
		subsystem:             "network",
		ignoredDevicesPattern: pattern,
		metricDescs:           map[string]*prometheus.Desc{},
		logger:                logger,
	}, nil
}

func (c *netClassCollector) Update(ch chan<- prometheus.Metric) error {
	return c.netClassSysfsUpdate(ch)
}

func (c *netClassCollector) netClassSysfsUpdate(ch chan<- prometheus.Metric) error {
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

func (c *netClassCollector) getFieldDesc(name string) *prometheus.Desc {
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

func getAdminState(flags *int64) string {
	if flags == nil {
		return "unknown"
	}

	if *flags&int64(net.FlagUp) == 1 {
		return "up"
	}

	return "down"
}
