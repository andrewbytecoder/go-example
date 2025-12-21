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

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	netclassIgnoredDevices = kingpin.Flag("collector.netclass.ignored-devices", "Regexp of net devices to ignore for netclass collector.").Default("^$").String()
	netclassInvalidSpeed   = kingpin.Flag("collector.netclass.ignore-invalid-speed", "Ignore devices where the speed is invalid. This will be the default behavior in 2.x.").Bool()
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
	pattern := regexp.MustCompile(*netclassIgnoredDevices)
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

	infoDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, c.subsystem, "info"),
		"Non-numeric data from /sys/class/net/<iface>, value is always 1.",
		[]string{"device", "address", "broadcast", "duplex", "operstate", "adminstate", "ifalias"},
		nil,
	)
	infoValue := 1.0

	ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, infoValue, ifaceInfo.Name, ifaceInfo.Address, ifaceInfo.Broadcast, ifaceInfo.Duplex, ifaceInfo.OperState, getAdminState(ifaceInfo.Flags), ifaceInfo.IfAlias)

	pushMetric(ch, c.getFieldDesc("address_assign_type"), "address_assign_type", ifaceInfo.AddrAssignType, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("carrier"), "carrier", ifaceInfo.Carrier, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("carrier_changes_total"), "carrier_changes_total", ifaceInfo.CarrierChanges, prometheus.CounterValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("carrier_up_changes_total"), "carrier_up_changes_total", ifaceInfo.CarrierUpCount, prometheus.CounterValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("carrier_down_changes_total"), "carrier_down_changes_total", ifaceInfo.CarrierDownCount, prometheus.CounterValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("device_id"), "device_id", ifaceInfo.DevID, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("dormant"), "dormant", ifaceInfo.Dormant, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("flags"), "flags", ifaceInfo.Flags, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("iface_id"), "iface_id", ifaceInfo.IfIndex, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("iface_link"), "iface_link", ifaceInfo.IfLink, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("iface_link_mode"), "iface_link_mode", ifaceInfo.LinkMode, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("mtu_bytes"), "mtu_bytes", ifaceInfo.MTU, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("name_assign_type"), "name_assign_type", ifaceInfo.NameAssignType, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("net_dev_group"), "net_dev_group", ifaceInfo.NetDevGroup, prometheus.GaugeValue, ifaceInfo.Name)

	if ifaceInfo.Speed != nil {
		// Some devices return -1 if the speed is unknown.
		if *ifaceInfo.Speed >= 0 || !*netclassInvalidSpeed {
			speedBytes := int64(*ifaceInfo.Speed * 1000 * 1000 / 8)
			pushMetric(ch, c.getFieldDesc("speed_bytes"), "speed_bytes", speedBytes, prometheus.GaugeValue, ifaceInfo.Name)
		}
	}

	pushMetric(ch, c.getFieldDesc("transmit_queue_length"), "transmit_queue_length", ifaceInfo.TxQueueLen, prometheus.GaugeValue, ifaceInfo.Name)
	pushMetric(ch, c.getFieldDesc("protocol_type"), "protocol_type", ifaceInfo.Type, prometheus.GaugeValue, ifaceInfo.Name)

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
