package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"gopkg.in/tylerb/graceful.v1"
	"gopkg.in/yaml.v2"
	"strings"
)

var (
	DefaultTimeout  = time.Minute
	DefaultInterval = time.Minute * 5
)

// Query defines a SQL statement and parameters as well as configuration
// for the monitoring behavior and result comparison.
type Query struct {
	Name       string
	Path       string
	Driver     string
	Connection map[string]interface{}
	SQL        string
	Params     map[string]interface{}
	Interval   time.Duration
	Timeout    time.Duration
}

func decodeQueries(r io.Reader) (map[string]*Query, error) {
	queries := make(map[string]*Query)

	b, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(b, &queries); err != nil {
		return nil, err
	}
	for k, q := range queries {
		// Set name and path.
		if len(k) > 0 && k[0] == '/' {
			q.Name = k[1:]
			q.Path = k
		} else {
			q.Name = k
			q.Path = "/" + k
		}

		if q.Driver == "" {
			return nil, errors.New("driver is required")
		}

		if q.SQL == "" {
			return nil, errors.New("SQL statement required")
		}

		if q.Interval == 0 {
			q.Interval = DefaultInterval
		}

		if q.Timeout == 0 {
			q.Timeout = DefaultTimeout
		}
	}

	return queries, nil
}

func decodeQueriesInDir(path string) (map[string]*Query, error) {
	queries := make(map[string]*Query)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		fn := f.Name()
		if strings.HasSuffix(fn, ".yml") {
			fn := fmt.Sprintf("%s/%s", strings.TrimRight(path, "/"), fn)
			log.Println("Loading", fn)
			file, err := os.Open(fn)
			if err != nil {
				return nil, err
			}
			q, err := decodeQueries(file)
			if err != nil {
				return nil, err
			}
			for k, v := range q {
				if queries[k] != nil {
					return nil, errors.New(fmt.Sprintf("Query %s already defined", k))
				}
				queries[k] = v
			}

			file.Close()
		}
	}
	return queries, nil
}

func main() {
	var (
		host        string
		port        int
		service     string
		queriesFile string
		queryDir    string
	)

	flag.StringVar(&host, "host", "", "Host of the service.")
	flag.IntVar(&port, "port", 8080, "Port of the service.")
	flag.StringVar(&service, "service", "", "Query of SQL agent service.")
	flag.StringVar(&queriesFile, "queries", "queries.yml", "Path to file containing queries.")
	flag.StringVar(&queryDir, "queryDir", "", "Path to directory containing queries.")

	flag.Parse()

	if service == "" {
		log.Fatal("URL to SQL Agent service required.")
	}

	if queriesFile == "queries.yml" && queryDir != "" {
		queriesFile = ""
	}
	if queriesFile != "" && queryDir != "" {
		log.Fatal("You can specify either -queries or -queryDir")
	}

	var (
		err     error
		queries map[string]*Query
	)
	if queryDir != "" {
		queries, err = decodeQueriesInDir(queryDir)
	} else {
		// Read queries for request body.
		file, err := os.Open(queriesFile)

		if err != nil {
			log.Fatal(err)
		}

		queries, err = decodeQueries(file)

		file.Close()
	}
	if err != nil {
		log.Fatal(err)
	}

	// Wait group of queries.
	wg := new(sync.WaitGroup)
	wg.Add(len(queries))

	// Shared context. Close the cxt.Done channel to stop the workers.
	cxt, cancel := context.WithCancel(context.Background())

	var w *Worker

	mux := http.NewServeMux()

	for _, q := range queries {
		// Create a new worker and start it in its own goroutine.
		w = NewWorker(q)
		go w.Start(context.WithValue(cxt, "wg", wg), service)
	}

	// Register the handler.
	mux.Handle("/metrics", prometheus.Handler())

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("* Listening on %s...", addr)

	// Handles OS kill and interrupt.
	graceful.Run(addr, 5*time.Second, mux)

	log.Print("canceling workers")
	cancel()
	log.Print("waiting for workers to finish")
	wg.Wait()
}
