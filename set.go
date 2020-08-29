package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type record map[string]interface{}
type records []record

type metricStatus int

const (
	registered metricStatus = iota
	unregistered
)

// QueryResult contains query results
type QueryResult struct {
	Query  *Query
	Result map[string]prometheus.Gauge // Internally we represent each facet with a JSON-encoded string for simplicity
}

// NewQueryResult initializes a new metrics collector.
func NewQueryResult(q *Query) *QueryResult {
	r := &QueryResult{
		Query:  q,
		Result: make(map[string]prometheus.Gauge),
	}

	return r
}

func (r *QueryResult) generateMetricName(suffix string) string {
	metricName := r.Query.Name
	if suffix != "" {
		metricName = fmt.Sprintf("%s_%s", r.Query.Name, suffix)
	}
	return metricName
}

func (r *QueryResult) generateMetricUniqueKey(facets map[string]interface{}, suffix string) string {
	jsonData, _ := json.Marshal(facets)
	return fmt.Sprintf("%s%s", r.generateMetricName(suffix), string(jsonData))
}

func (r *QueryResult) createMetric(facets map[string]interface{}, suffix string) (string, metricStatus) {
	metricName := r.generateMetricName(suffix)
	resultKey := r.generateMetricUniqueKey(facets, suffix)

	labels := prometheus.Labels{}
	for k, v := range facets {
		labels[k] = strings.ToLower(fmt.Sprintf("%v", v))
	}

	if _, ok := r.Result[resultKey]; ok {
		// A metric with this key is already created and assumed to be registered
		return resultKey, registered
	}

	fmt.Println("Creating", resultKey)
	r.Result[resultKey] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("query_result_%s", metricName),
		Help:        "Result of an SQL query",
		ConstLabels: labels,
	})
	return resultKey, unregistered
}

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

// SetMetrics set and register metrics
func (r *QueryResult) SetMetrics(recs records, valueOnError string) error {
	// Queries that return only one record should only have one column
	if len(recs) > 1 && len(recs[0]) == 1 {
		return errors.New("There is more than one row in the query result - with a single column")
	}

	if r.Query.DataField != "" && len(r.Query.SubMetrics) > 0 {
		return errors.New("sub-metrics are not compatible with data-field")
	}

	// We need to make sure not to default to a value on error before
	// it has been registered once before since re-registering might
	// not work if different labels are used.
	if len(recs) == 0 && valueOnError != "" {
		metricSet := false
		for k := range r.Result {
			if strings.HasPrefix(k, r.generateMetricName("")) {
				err := setValueForResult(r.Result[k], valueOnError)
				if err != nil {
					return err
				}
				metricSet = true
			}
		}
		if metricSet {
			return nil
		}
	}

	submetrics := map[string]string{}

	if len(r.Query.SubMetrics) > 0 {
		submetrics = r.Query.SubMetrics
	} else {
		submetrics = map[string]string{"": r.Query.DataField}
	}

	facetsWithResult := make(map[string]metricStatus, 0)
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
						return errors.New("Data field not specified for multi-column query")
					}
					dataVal = v
					dataFound = true
				}
			}

			if !dataFound {
				return errors.New("Data field not found in result set")
			}

			key, status := r.createMetric(facet, suffix)
			err := setValueForResult(r.Result[key], dataVal)
			if err != nil {
				return err
			}
			facetsWithResult[key] = status
		}
	}
	r.registerMetrics(facetsWithResult)
	return nil
}

// RegisterMetrics registers and unregister gauges
func (r *QueryResult) registerMetrics(facetsWithResult map[string]metricStatus) {
	for key, m := range r.Result {
		status, ok := facetsWithResult[key]
		if !ok {
			fmt.Println("Unregistering metric", key)
			prometheus.Unregister(m)
			delete(r.Result, key)
			continue
		}
		if status == unregistered {
			defer func(key string, m prometheus.Gauge) {
				fmt.Println("Registering metric", key)
				prometheus.MustRegister(m)
			}(key, m)
		}
	}
}
