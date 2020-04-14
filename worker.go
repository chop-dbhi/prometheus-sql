package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"golang.org/x/net/context"
)

// Backoff for fetching. It starts by waiting the minimum duration after a
// failed fetch, doubling it each time (with a bitter of jitter) up to max
// duration between requests.
var defaultBackoff = backoff.Backoff{
	Min:    1 * time.Second,
	Max:    5 * time.Minute,
	Jitter: true,
	Factor: 2,
}

type Worker struct {
	query       *Query
	payload     []byte
	client      *http.Client
	result      *QueryResult
	log         *log.Logger
	backoff     backoff.Backoff
	ctx         context.Context
	lastRecords []record // store last successfull records (will be used in case of error)
}

func (w *Worker) SetMetrics(recs records) {
	list, err := w.result.SetMetrics(recs)
	if err != nil {
		w.log.Printf("Error setting metrics: %s", err)
		return
	}

	w.result.RegisterMetrics(list)
	w.lastRecords = recs
}

func (w *Worker) Fetch(url string) (records, error) {
	var (
		t    time.Time
		err  error
		req  *http.Request
		resp *http.Response
	)

	for {
		t = time.Now()

		req, err = http.NewRequest("POST", url, bytes.NewBuffer(w.payload))

		if err != nil {
			panic(err)
		}
		req = req.WithContext(w.ctx)

		// Set the content-type of the request body and accept LD-JSON.
		req.Header.Set("content-type", "application/json")
		req.Header.Set("accept", "application/json")

		resp, err = w.client.Do(req)

		// No formal error, but a non-successful status code. Construct an error.
		if err == nil && resp.StatusCode != 200 {
			b, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			err = fmt.Errorf("%s: %s", resp.Status, string(b))
		}

		// No error, break to read the data.
		if err == nil {
			break
		}

		w.setMetricsValueOnError()

		// Backoff on an error.
		w.log.Print(err)
		d := w.backoff.Duration()
		w.log.Printf("Backing off for %s", d)
		select {
		case <-time.After(d):
			continue
		case <-w.ctx.Done():
			return nil, errors.New("Execution was canceled")
		}
	}

	w.backoff.Reset()

	w.log.Printf("Fetch took %s", time.Now().Sub(t))

	var recs []record

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&recs); err != nil {
		return nil, err
	}

	w.SetMetrics(recs)

	return recs, nil
}

func (w *Worker) Start(url string) {
	tick := func() {
		_, err := w.Fetch(url)
		if err != nil {
			w.log.Printf("Error fetching records: %s", err)
			return
		}
	}

	tick()
	ticker := time.NewTicker(w.query.Interval)

	for {
		select {
		case <-w.ctx.Done():
			wg, _ := w.ctx.Value("wg").(*sync.WaitGroup)
			wg.Done()
			w.log.Printf("Stopping worker")
			return

		case <-ticker.C:
			tick()
		}
	}
}

// Set metrics with the provided value-on-error
func (w *Worker) setMetricsValueOnError() {
	// skip if value-on-error is not set
	if w.query.ValueOnError != "" {
		return
	}

	if w.lastRecords == nil {
		w.log.Printf("Could not set error value becaue there is no previous record")
		return
	}

	// update value
	for _, record := range w.lastRecords {
		record["value"] = w.query.ValueOnError
	}

	w.SetMetrics(w.lastRecords)
}

// NewWorker creates a new worker for a query.
func NewWorker(ctx context.Context, q *Query) *Worker {
	// Encode the payload once for all subsequent requests.
	payload, err := json.Marshal(map[string]interface{}{
		"driver":     q.Driver,
		"connection": q.Connection,
		"sql":        q.SQL,
		"params":     q.Params,
	})

	if err != nil {
		panic(err)
	}

	return &Worker{
		query:   q,
		result:  NewQueryResult(q),
		payload: payload,
		backoff: defaultBackoff,
		log:     log.New(os.Stderr, fmt.Sprintf("[%s] ", q.Name), log.LstdFlags),
		client: &http.Client{
			Timeout: q.Timeout,
		},
		ctx: ctx,
	}
}
