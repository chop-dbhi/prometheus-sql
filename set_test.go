package main

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
)

type testQuerySetOptions struct {
	q       *QueryResult
	rec     records
	results map[string]string
}

func TestSimpleQuerySet(t *testing.T) {
	(&testQuerySetOptions{
		q: NewQueryResult(&Query{
			Name:      "simple_metric",
			DataField: "value",
		}),
		rec: records{
			record{
				"name":  "foo",
				"value": 67,
			},
		},
		results: map[string]string{
			`simple_metric{"name":"foo"}`: `label: <
  name: "name"
  value: "foo"
>
gauge: <
  value: 67
>
`,
		},
	}).testQuerySet(t)
}

func TestSingleQuerySet(t *testing.T) {
	(&testQuerySetOptions{
		q: NewQueryResult(&Query{
			Name: "single_metric",
		}),
		rec: records{
			record{
				"value": 42,
			},
		},
		results: map[string]string{
			"single_metric{}": `gauge: <
  value: 42
>
`,
		},
	}).testQuerySet(t)
}

func TestSubMetrics(t *testing.T) {
	(&testQuerySetOptions{
		q: NewQueryResult(&Query{
			Name: "submetrics_metric",
			SubMetrics: map[string]string{
				"total": "rt",
				"count": "cnt",
			},
		}),
		rec: records{
			record{
				"rt":   200,
				"cnt":  5,
				"name": "foo",
			},
			record{
				"rt":   500,
				"cnt":  3,
				"name": "bar",
			},
		},
		results: map[string]string{
			`submetrics_metric_total{"name":"foo"}`: `label: <
  name: "name"
  value: "foo"
>
gauge: <
  value: 200
>
`,
			`submetrics_metric_count{"name":"foo"}`: `label: <
  name: "name"
  value: "foo"
>
gauge: <
  value: 5
>
`,
			`submetrics_metric_total{"name":"bar"}`: `label: <
  name: "name"
  value: "bar"
>
gauge: <
  value: 500
>
`,
			`submetrics_metric_count{"name":"bar"}`: `label: <
  name: "name"
  value: "bar"
>
gauge: <
  value: 3
>
`,
		},
	}).testQuerySet(t)
}

func (opts *testQuerySetOptions) testQuerySet(t *testing.T) {
	_, err := opts.q.SetMetrics(opts.rec)
	if err != nil {
		t.Errorf("Error while setting metrics: %v", err)
		return
	}

	numRes := len(opts.q.Result)
	if numRes != len(opts.results) {
		t.Errorf("Bad number of result ; expected: %d, got: %d.", numRes, len(opts.results))
		return
	}

	for k, v := range opts.results {
		m := opts.q.Result[k]
		if m == nil {
			t.Errorf("Can not find metric `%s`.", k)
			return
		}
		metric := &dto.Metric{}
		m.Write(metric)
		out := proto.MarshalTextString(metric)
		if v != out {
			t.Errorf("[%s] bad output ; expected: \n`%s`\nGot: \n`%s`\n", k, v, out)
			return
		}
	}

}
