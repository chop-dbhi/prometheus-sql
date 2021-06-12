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

// Worker is responsible for fetching data via SQL Agent
type Worker struct {
	query   *Query
	payload []byte
	client  *http.Client
	result  *QueryResult
	log     *log.Logger
	backoff backoff.Backoff
	ctx     context.Context
}

func (w *Worker) setQueryResultMetrics(recs records) {
	err := w.result.SetMetrics(recs, w.query.ValueOnError)
	if err != nil {
		w.log.Printf("Error setting metrics: %s", err)
		return
	}
}

func (w *Worker) queryResultError() {
	w.setQueryResultMetrics(nil)
}

func (w *Worker) fetchRecords(url string) error {
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
		w.log.Print(err)

		w.queryResultError()

		// Backoff on an error.
		d := w.backoff.Duration()
		w.log.Printf("Backing off for %s", d)
		select {
		case <-time.After(d):
			continue
		case <-w.ctx.Done():
			return errors.New("Execution was canceled")
		}
	}

	w.backoff.Reset()

	w.log.Printf("Fetch took %s", time.Now().Sub(t))

	var recs []record

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&recs); err != nil {
		return err
	}

	w.setQueryResultMetrics(recs)

	return nil
}

// Start fetching data from specified URL
func (w *Worker) Start(url string, wg *sync.WaitGroup) {
	tick := func() {
		err := w.fetchRecords(url)
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
			wg.Done()
			w.log.Printf("Stopping worker")
			return

		case <-ticker.C:
			tick()
		}
	}
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
