package main

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type SetMetrics struct {
	sync.Mutex

	// Absolute metrics across states.
	Size prometheus.Gauge

	// Metrics relative to the previous state.
	Additions prometheus.Counter
	Removals  prometheus.Counter
	Changes   prometheus.Counter
}

func (m *SetMetrics) Register() {
	prometheus.MustRegister(m.Size)
	prometheus.MustRegister(m.Additions)
	prometheus.MustRegister(m.Removals)
	prometheus.MustRegister(m.Changes)
}

func NewSetMetrics() *SetMetrics {
	return &SetMetrics{
		Size: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "set_size",
			Help: "Size of the set.",
		}),

		Additions: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "set_additions",
			Help: "Number of additions to the set.",
		}),

		Removals: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "set_removals",
			Help: "Number of removals from the set.",
		}),

		Changes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "set_changes",
			Help: "Number of item that changed in the set.",
		}),
	}
}
