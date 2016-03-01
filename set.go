package main

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
)

// SetMetrics is an interface for tracking metrics for a given set.
type SetMetrics struct {
	Name string

	Size      prometheus.Gauge
	Additions prometheus.Gauge
	Removals  prometheus.Gauge
	Changes   prometheus.Gauge
}

// Compares the two states and writes the metric values. The first set value
// denotes the previous state. For the first evaluation this should be nil.
// The new state (b) should always be non-nil.
func (m *SetMetrics) Compare(a, b *Set) {
	var adds, chgs, rms int

	// Compare against previous state.
	if a != nil {
		var (
			ok     bool
			ak, bk string
			av, bv interface{}
		)

		for ak, av = range a.index {
			if bv, ok = b.index[ak]; !ok {
				rms++
				continue
			}

			if !reflect.DeepEqual(av, bv) {
				chgs++
			}
		}

		for bk, _ = range b.index {
			if _, ok = a.index[bk]; !ok {
				adds++
			}
		}
	}

	m.Size.Set(float64(b.Size()))
	m.Additions.Set(float64(adds))
	m.Removals.Set(float64(rms))
	m.Changes.Set(float64(chgs))
}

// NewSetMetrics initializes a new metrics collector.
func NewSetMetrics(name string) *SetMetrics {
	labels := prometheus.Labels{
		"query": name,
	}

	m := &SetMetrics{
		Name: name,

		Size: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "set_size",
			Help:        "Size of the set.",
			ConstLabels: labels,
		}),

		Additions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "set_additions",
			Help:        "Number of additions to the set.",
			ConstLabels: labels,
		}),

		Removals: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "set_removals",
			Help:        "Number of removals from the set.",
			ConstLabels: labels,
		}),

		Changes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "set_changes",
			Help:        "Number of item that changed in the set.",
			ConstLabels: labels,
		}),
	}

	prometheus.MustRegister(m.Size)
	prometheus.MustRegister(m.Additions)
	prometheus.MustRegister(m.Removals)
	prometheus.MustRegister(m.Changes)

	return m
}

type record map[string]interface{}

type Set struct {
	key     string
	records []record
	index   map[string]record
}

var (
	ErrDuplicateItem     = errors.New("duplicate item found for set")
	ErrInvalidIdentifier = errors.New("identifier is not a colum")
)

func NewSet(key string, records []record) (*Set, error) {
	s := &Set{
		key:     key,
		records: records,
	}

	s.index = make(map[string]record, len(records))

	var (
		idv interface{}
		id  string
	)

	for _, r := range records {
		idv = r[key]

		if idv == nil {
			return nil, ErrInvalidIdentifier
		}

		id = fmt.Sprint(idv)

		if _, ok := s.index[id]; ok {
			return nil, ErrDuplicateItem
		}

		s.index[id] = r
	}

	return s, nil
}

func (s *Set) Size() int {
	return len(s.records)
}
