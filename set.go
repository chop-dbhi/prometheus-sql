package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type QueryResult struct {
	Query  *Query
	Result map[string]prometheus.Gauge // Internally we represent each facet with a JSON-encoded string for simplicity
}

// NewSetMetrics initializes a new metrics collector.
func NewQueryResult(q *Query) *QueryResult {
	r := &QueryResult{
		Query:  q,
		Result: make(map[string]prometheus.Gauge),
	}

	return r
}

func (r *QueryResult) registerMetric(facets map[string]interface{}, suffix string) string {
	labels := prometheus.Labels{}
	metricName := r.Query.Name
	if suffix != "" {
		metricName = fmt.Sprintf("%s_%s", r.Query.Name, suffix)
	}

	jsonData, _ := json.Marshal(facets)
	resultKey := fmt.Sprintf("%s%s", metricName, string(jsonData))

	for k, v := range facets {
		labels[k] = strings.ToLower(fmt.Sprintf("%v", v))
	}

	if _, ok := r.Result[resultKey]; ok { // A metric with this name is already registered
		return resultKey
	}

	fmt.Println("Registering metric", resultKey)
	r.Result[resultKey] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("query_result_%s", metricName),
		Help:        "Result of an SQL query",
		ConstLabels: labels,
	})
	prometheus.MustRegister(r.Result[resultKey])
	return resultKey
}

type record map[string]interface{}
type records []record

func setValueForResult(r prometheus.Gauge, v interface{}) error {
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

func (r *QueryResult) SetMetrics(recs records) (map[string]bool, error) {
	// Queries that return only one record should only have one column
	if len(recs) > 1 && len(recs[0]) == 1 {
		return nil, errors.New("There is more than one row in the query result - with a single column")
	}

	if r.Query.DataField != "" && len(r.Query.SubMetrics) > 0 {
		return nil, errors.New("sub-metrics are not compatible with data-field")
	}

	submetrics := map[string]string{}

	if len(r.Query.SubMetrics) > 0 {
		submetrics = r.Query.SubMetrics
	} else {
		submetrics = map[string]string{"": r.Query.DataField}
	}

	facetsWithResult := make(map[string]bool, 0)
	for _, row := range recs {
		for suffix, datafield := range submetrics {
			facet := make(map[string]interface{})
			var (
				dataVal   interface{}
				dataFound bool
			)
			for k, v := range row {
				if len(row) > 1 && strings.ToLower(k) != datafield { // facet field, add to facets
					submetric := false
					for _, n := range submetrics {
						if strings.ToLower(k) == n {
							submetric = true
						}
					}
					// it is a facet field and not a submetric field
					if !submetric {
						facet[strings.ToLower(fmt.Sprintf("%v", k))] = v
					}
				} else { // this is the actual gauge data
					if dataFound {
						return nil, errors.New("Data field not specified for multi-column query")
					}
					dataVal = v
					dataFound = true
				}
			}

			if !dataFound {
				return nil, errors.New("Data field not found in result set")
			}

			key := r.registerMetric(facet, suffix)
			err := setValueForResult(r.Result[key], dataVal)
			if err != nil {
				return nil, err
			}
			facetsWithResult[key] = true
		}
	}

	return facetsWithResult, nil
}

func (r *QueryResult) DealWithMissingMetrics(facetsWithResult map[string]bool) {
	for key, m := range r.Result {
		if _, ok := facetsWithResult[key]; ok {
			continue
		}
		if r.Query.ReplaceMissing {
			fmt.Printf("Resetting metric %v to %v\n", key, r.Query.ReplaceValue)
			m.Set(r.Query.ReplaceValue)
		} else {
			fmt.Println("Unregistering metric", key)
			prometheus.Unregister(m)
			delete(r.Result, key)
		}
	}
}
