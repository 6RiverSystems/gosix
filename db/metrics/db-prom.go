package metrics

import (
	"database/sql"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func cachedDbStats(
	db *sql.DB,
	ttl time.Duration,
) func() *sql.DBStats {
	m := &sync.Mutex{}
	var stats *sql.DBStats
	var nextUpdate time.Time
	return func() *sql.DBStats {
		m.Lock()
		defer m.Unlock()
		now := time.Now()
		if stats == nil || now.After(nextUpdate) {
			stats_ := db.Stats()
			stats = &stats_
			nextUpdate = now.Add(ttl)
		}
		return stats
	}
}

func MonitorDB(
	db *sql.DB,
	labels map[string]string,
	ttl time.Duration,
) {
	cachedStats := cachedDbStats(db, ttl)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_max_open_connections",
			Help:        "Maximum number of open connections to the database.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().MaxOpenConnections)
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_open_connections",
			Help:        "The number of established connections both in use and idle.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().OpenConnections)
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_in_use",
			Help:        "The number of connections currently in use.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().InUse)
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_idle",
			Help:        "The number of idle connections.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().Idle)
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_wait_count",
			Help:        "The total number of connections waited for.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().WaitCount)
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_wait_duration",
			Help:        "The total time blocked waiting for a new connection.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().WaitDuration.Seconds())
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_max_idle_closed",
			Help:        "The total number of connections closed due to SetMaxIdleConns.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().MaxIdleClosed)
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:        "dbstats_max_lifetime_closed",
			Help:        "The total number of connections closed due to SetConnMaxLifetime.",
			ConstLabels: labels,
		},
		func() float64 {
			return float64(cachedStats().MaxLifetimeClosed)
		},
	)
}
