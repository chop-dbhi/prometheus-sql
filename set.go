package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"strconv"
	"strings"
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

// NewQueryMetrics initializes a new metrics collector.
func NewQueryResult(q *Query) *QueryResult {
	r := &QueryResult{
		Query:  q,
		Result: make(map[string]prometheus.Gauge),
	}

	return r
}

func (r *QueryResult) registerMetric(facets map[string]interface{}, suffix string, valueCase string, helpText string) (string, metricStatus) {
	labels := prometheus.Labels{}
	metricName := r.Query.Name
	if suffix != "" {
		metricName = fmt.Sprintf("%s_%s", r.Query.Name, suffix)
	}

	jsonData, _ := json.Marshal(facets)
	resultKey := fmt.Sprintf("%s%s", metricName, string(jsonData))

	for k, v := range facets {
		labels[k] = CaseChange(fmt.Sprintf("%v", v), valueCase)
	}

	if _, ok := r.Result[resultKey]; ok {
		// A metric with this key is already created and assumed to be registered
		return resultKey, registered
	}

	if len(helpText) == 0 {
		helpText = "Result of an SQL query"
	}

	fmt.Println("Creating", resultKey)
	r.Result[resultKey] = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        fmt.Sprintf("query_result_%s", metricName),
		Help:        helpText,
		ConstLabels: labels,
	})
	return resultKey, unregistered
}

func setValueForResult(r prometheus.Gauge, v interface{}) error {
	switch t := v.(type) {
	case nil:
		r.Set(math.NaN())
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

func (r *QueryResult) SetMetrics(recs records) (map[string]metricStatus, error) {
	// Queries that return only one record should only have one column
	if len(recs) > 1 && len(recs[0]) == 1 {
		return nil, errors.New("There is more than one row in the query result - with a single column")
	}

	if r.Query.DataField != "" && len(r.Query.SubMetrics) > 0 {
		return nil, errors.New("sub-metrics are not compatible with data-field")
	}

	submetrics := map[string]string{}
	extralabels := map[string]string{}
	labelCase := r.Query.LabelCase
	valueCase := r.Query.ValueCase

	if len(r.Query.SubMetrics) > 0 {
		submetrics = r.Query.SubMetrics
	} else {
		submetrics = map[string]string{"": r.Query.DataField}
	}

	if len(r.Query.ExtraLabels) > 0 {
		extralabels = r.Query.ExtraLabels
	}

	facetsWithResult := make(map[string]metricStatus, 0)
	for _, row := range recs {
		for suffix, datafield := range submetrics {
			facet := make(map[string]interface{})
			for k, v := range extralabels {
				facet[k] = v
			}
			var (
				dataVal   interface{}
				dataFound bool
			)
			datafield = CaseChange(datafield, labelCase)
			histogram_data := make(map[string]interface{})
			histogram := (datafield[len(datafield)-1:] == "#")
			for k, v := range row {
				if len(row) > 1 && k != datafield {
					k := CaseChange(fmt.Sprintf("%v", k), labelCase)
					if histogram && strings.HasPrefix(k, datafield) {
						// histogram field, add to histogram_data
						histogram_data[k[len(datafield):]] = v
						dataFound = true
					} else {
						// facet field, add to facets
						submetric := false
						for _, n := range submetrics {
							if k == CaseChange(n, labelCase) {
								submetric = true
							} else if strings.Contains(n, "#") && strings.HasPrefix(k, CaseChange(n, labelCase)) {
								submetric = true
							}
						}
						// it is a facet field and not a submetric field
						if !submetric {
							facet[k] = v
						}
					}
				} else { // this is the actual gauge data
					if dataFound {
						return nil, errors.New(fmt.Sprintf("Data field '%v' not specified for multi-column query", datafield))
					}
					dataVal = v
					dataFound = true
				}
			}

			if !dataFound {
				return nil, errors.New(fmt.Sprintf("Data field '%v' not found in result set", datafield))
			}

			if histogram {
				histogram_field := datafield[0 : len(datafield)-1]
				for k, dataVal := range histogram_data {
					// loop over histogram data registering bins
					facet[histogram_field] = k

					key, status := r.registerMetric(facet, suffix, valueCase, r.Query.HelpText)
					err := setValueForResult(r.Result[key], dataVal)
					if err != nil {
						return nil, err
					}
					facetsWithResult[key] = status
				}
			} else {
				key, status := r.registerMetric(facet, suffix, valueCase, r.Query.HelpText)
				err := setValueForResult(r.Result[key], dataVal)
				if err != nil {
					return nil, err
				}
				facetsWithResult[key] = status
			}
		}
	}

	return facetsWithResult, nil
}

// RegisterMetrics registers and unregister gauges
func (r *QueryResult) RegisterMetrics(facetsWithResult map[string]metricStatus) {
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
func CaseChange(str string, newCase string) string {
	switch newCase {
	case "lower":
		return strings.ToLower(str)
	case "upper":
		return strings.ToUpper(str)
	case "first":
		return string(strings.ToLower(str[0:1])) + str[1:]
	case "title":
		return string(strings.ToUpper(str[0:1])) + str[1:]
	case "keep":
		return str
	}
	return strings.ToLower(str)
}
