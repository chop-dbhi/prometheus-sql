package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
)

// In-memory cache of previous state of the records so the metrics
// can be produced.
var (
	metrics = NewSetMetrics()

	previousState *Set
)

func init() {
	metrics.Register()
}

type config struct {
	Driver     string
	Connection map[string]interface{}
	SQL        string
	Params     map[string]interface{}
}

func yamlToJSON(r io.Reader) ([]byte, error) {
	var (
		err error
		b   []byte
		buf bytes.Buffer

		config config
	)

	if b, err = ioutil.ReadAll(r); err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	if err = json.NewEncoder(&buf).Encode(config); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func getState(key string, url string, body []byte) (*Set, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("service unreachable")
	}

	var records []record

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}

	return NewSet(key, records)
}

func main() {
	var (
		host       string
		port       int
		service    string
		options    string
		interval   time.Duration
		identifier string
	)

	flag.StringVar(&host, "host", "", "Host of the service.")
	flag.IntVar(&port, "port", 8080, "Port of the service.")
	flag.StringVar(&service, "service", "", "Endpoint of SQL agent service.")
	flag.StringVar(&options, "options", "sql.yml", "Path to file containing SQL agent options.")
	flag.DurationVar(&interval, "interval", time.Hour, "Interval of pull.")
	flag.StringVar(&identifier, "identifier", "", "Column name of the unique identifier.")

	flag.Parse()

	if service == "" {
		log.Fatal("endpoint to SQL agent service required.")
	}

	if identifier == "" {
		log.Fatal("identifier column required.")
	}

	// Read options for request body.
	file, err := os.Open(options)

	if err != nil {
		log.Fatal(err)
	}

	payload, err := yamlToJSON(file)
	file.Close()

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		var err error

		t0 := time.Now()

		log.Print("Fetching initial state...")

		previousState, err = getState(identifier, service, payload)

		if err != nil {
			log.Fatalf("Error fetching state: %s", err)
		}

		log.Printf("Fetch took %v", time.Now().Sub(t0))

		ticker := time.NewTicker(interval)

		for {
			select {
			case <-ticker.C:
				log.Print("Fetching state...")
				t0 = time.Now()

				newState, err := getState(identifier, service, payload)

				if err != nil {
					log.Printf("Error fetching state: %s", err)
					break
				}

				log.Printf("Fetch took %v", time.Now().Sub(t0))

				previousState.Compare(metrics, newState)
				previousState = newState

				log.Printf("Logged metrics")
			}
		}
	}()

	http.Handle("/metrics", prometheus.Handler())

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("* Listening on %s...", addr)

	http.ListenAndServe(addr, nil)
}
