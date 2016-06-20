package main

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type QueryResult struct {
	Name   string
	Result prometheus.Counter
}

// NewSetMetrics initializes a new metrics collector.
func NewQueryResult(name string) *QueryResult {
	labels := prometheus.Labels{
		"query": name,
	}

	r := &QueryResult{
		Name: name,

		Result: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "query_result",
			Help:        "Result of an SQL query",
			ConstLabels: labels,
		}),
	}

	prometheus.MustRegister(r.Result)

	return r
}

type record map[string]interface{}
type records []record

func (r *QueryResult) SetMetrics(recs records) {

	for _, v := range recs[0] {

		if s, ok := v.(string); ok {
			f, err := strconv.ParseFloat(s, 64)
			if err == nil {
				r.Result.Set(f)
			}
		}
	}
}
