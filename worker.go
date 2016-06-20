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
	query   *Query
	payload []byte
	client  *http.Client
	result  *QueryResult
	log     *log.Logger
	backoff backoff.Backoff
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

		// Backoff on an error.
		w.log.Print(err)
		d := w.backoff.Duration()
		w.log.Printf("Backing off for %s", d)
		time.Sleep(d)
	}

	w.backoff.Reset()

	w.log.Printf("Fetch took %s", time.Now().Sub(t))

	var recs []record

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&recs); err != nil {
		return nil, err
	}

	// queries should return only one record
	if len(recs) > 1 {
		return nil, errors.New("There is more than one line in the query result")
	}
	if len(recs[0]) > 1 {
		return nil, errors.New("There is more than one column in the query result")
	}

	return recs, nil
}

func (w *Worker) Start(cxt context.Context, url string) {
	recs, err := w.Fetch(url)
	if err != nil {
		w.log.Printf("Error fetching records: %s", err)
	}

	w.result.SetMetrics(recs)

	ticker := time.NewTicker(w.query.Interval)

	for {
		select {
		case <-cxt.Done():
			wg, _ := cxt.Value("wg").(*sync.WaitGroup)
			wg.Done()
			w.log.Printf("Stopping worker")
			return

		case <-ticker.C:
			if recs, err = w.Fetch(url); err != nil {
				w.log.Printf("Error fetching records: %s", err)
				break
			}

			// Compare with new state and log metrics.
			w.result.SetMetrics(recs)
		}
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
		result:  NewQueryResult(q.Name),
		payload: payload,
		backoff: defaultBackoff,
		log:     log.New(os.Stderr, fmt.Sprintf("[%s] ", q.Name), log.LstdFlags),
		client: &http.Client{
			Timeout: q.Timeout,
		},
	}
}
