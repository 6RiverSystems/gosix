package faults

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// TODO: to refactor this to gosix, we'll need a way to inject the AppName.
// May need to make the fault set part of the registry.

type ActiveFaultsCollector struct {
	set *Set
}

func NewActiveFaultsCollector(s *Set) *ActiveFaultsCollector {
	return &ActiveFaultsCollector{s}
}

func (c *ActiveFaultsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.set.activeFaultsDesc
	ch <- c.set.remainingFaultsDesc
}

func (c *ActiveFaultsCollector) Collect(ch chan<- prometheus.Metric) {
	// we could use Set.Current(), but we can avoid a lot of copying with custom
	// code here
	c.set.mu.RLock()
	defer c.set.mu.RUnlock()
	for op, l := range c.set.faults {
		active := 0
		remaining := int64(0)
		for _, d := range l {
			if r := atomic.LoadInt64(&d.Count); r > 0 {
				active++
				remaining += r
			}
		}
		ch <- prometheus.MustNewConstMetric(
			c.set.activeFaultsDesc,
			prometheus.GaugeValue,
			float64(active),
			op,
		)
		ch <- prometheus.MustNewConstMetric(
			c.set.remainingFaultsDesc,
			prometheus.GaugeValue,
			float64(remaining),
			op,
		)
	}
}
