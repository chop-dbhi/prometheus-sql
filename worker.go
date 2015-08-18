package main

import (
	"bytes"
	"encoding/json"
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
	query   *Query
	state   *Set
	payload []byte
	client  *http.Client
	metrics *SetMetrics
	log     *log.Logger
	backoff backoff.Backoff
}

func (w *Worker) Fetch(url string) (*Set, error) {
	var (
		t    time.Time
		err  error
		resp *http.Response
	)

	for {
		t = time.Now()

		resp, err = w.client.Post(url, "application/json", bytes.NewBuffer(w.payload))

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

		// Backoff on an error.
		w.log.Print(err)
		d := w.backoff.Duration()
		w.log.Printf("Backing off for %s", d)
		time.Sleep(d)
	}

	w.backoff.Reset()

	w.log.Printf("Fetch took %s", time.Now().Sub(t))

	var records []record

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}

	return NewSet(w.query.Identifier, records)
}

func (w *Worker) Start(cxt context.Context, url string) {
	state, err := w.Fetch(url)

	if err != nil {
		w.log.Printf("Error fetching state: %s", err)
	}

	// Set the initial state.
	w.state = state

	if state != nil {
		w.metrics.Compare(nil, state)
	}

	ticker := time.NewTicker(w.query.Interval)

	for {
		select {
		case <-cxt.Done():
			wg, _ := cxt.Value("wg").(*sync.WaitGroup)
			wg.Done()
			w.log.Printf("Stopping worker")
			return

		case <-ticker.C:
			if state, err = w.Fetch(url); err != nil {
				w.log.Printf("Error fetching state: %s", err)
				break
			}

			// Compare with new state and log metrics.
			w.metrics.Compare(w.state, state)
			w.state = state
		}
	}
}

// StateHandler returns an http.HandlerFunc that writes the state of the query.
func (w *Worker) StateHandler() http.HandlerFunc {
	return func(rp http.ResponseWriter, _ *http.Request) {
		if w.state == nil {
			rp.WriteHeader(http.StatusNoContent)
			return
		}

		if err := json.NewEncoder(rp).Encode(w.state.records); err != nil {
			rp.WriteHeader(http.StatusInternalServerError)
			rp.Write([]byte(err.Error()))
			return
		}

		rp.Header().Set("content-type", "application/json")
	}
}

// NewWorker creates a new worker for a query.
func NewWorker(q *Query) *Worker {
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
		metrics: NewSetMetrics(q.Name),
		payload: payload,
		backoff: defaultBackoff,
		log:     log.New(os.Stderr, fmt.Sprintf("[%s] ", q.Name), log.LstdFlags),
		client: &http.Client{
			Timeout: q.Timeout,
		},
	}
}
