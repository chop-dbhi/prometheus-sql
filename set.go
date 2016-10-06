package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
)

type QueryResult struct {
	Query  *Query
	Result map[string]prometheus.Counter // Internally we represent each facet with a JSON-encoded string for simplicity
}

// NewSetMetrics initializes a new metrics collector.
func NewQueryResult(q *Query) *QueryResult {
	r := &QueryResult{
		Query: q,
	}
	r.Result = make(map[string]prometheus.Counter)

	return r
}

func (r *QueryResult) registerMetric(facets map[string]interface{}) string {
	labels := prometheus.Labels{}

	jsonData, _ := json.Marshal(facets)
	resultKey := string(jsonData)

	for k, v := range facets {
		labels[k] = strings.ToLower(fmt.Sprintf("%v", v))
	}

	if _, ok := r.Result[resultKey]; ok { // A metric with this name is already registered
		return resultKey
	}

	fmt.Println("Registering metric", r.Query.Name, "with facets", resultKey)
	r.Result[resultKey] = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        fmt.Sprintf("query_result_%s", r.Query.Name),
		Help:        "Result of an SQL query",
		ConstLabels: labels,
	})
	prometheus.MustRegister(r.Result[resultKey])
	return resultKey
}

type record map[string]interface{}
type records []record

func setValueForResult(r prometheus.Counter, v interface{}) error {
	switch t := v.(type) {
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return err
		}
		r.Set(f)
	case int:
		r.Set(float64(t))
	case float64:
		r.Set(t)
	default:
		return fmt.Errorf("Unhandled type %s", t)
	}
	return nil
}

func (r *QueryResult) SetMetrics(recs records) error {
	// Queries that return only one record should only have one column
	if len(recs) > 1 && len(recs[0]) == 1 {
		return errors.New("There is more than one row in the query result - with a single column")
	}

	for _, row := range recs {
		facet := make(map[string]interface{})
		var (
			dataVal   interface{}
			dataFound bool
		)
		if len(row) > 1 && r.Query.DataField == "" {
			return errors.New("Data field not specified for multi-column query")
		}
		for k, v := range row {
			if len(row) > 1 && strings.ToLower(k) != r.Query.DataField { // facet field, add to facets
				facet[strings.ToLower(fmt.Sprintf("%v", k))] = v
			} else { // this is the actual counter data
				dataVal = v
				dataFound = true
			}
		}

		if !dataFound {
			return errors.New("Data field not found in result set")
		}

		key := r.registerMetric(facet)
		err := setValueForResult(r.Result[key], dataVal)
		if err != nil {
			return err
		}
	}

	return nil
}
