package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
	"gopkg.in/tylerb/graceful.v1"
)

func main() {
	log.Println("prometheus-sql starting up...")
	var (
		host                         string
		port                         int
		service                      string
		queriesFile                  string
		queryDir                     string
		confFile                     string
		tolerateInvalidQueryDirFiles bool
	)

	flag.StringVar(&host, "host", DefaultHost, "Host of the service.")
	flag.IntVar(&port, "port", DefaultPort, "Port of the service.")
	flag.StringVar(&service, "service", DefaultService, "Query of SQL agent service.")
	flag.StringVar(&queriesFile, "queries", DefaultQueriesFile, "Path to file containing queries.")
	flag.StringVar(&queryDir, "queryDir", DefaultQueriesDir, "Path to directory containing queries.")
	flag.StringVar(&confFile, "config", DefaultConfFile, "Configuration file to define common data sources etc.")
	flag.BoolVar(&tolerateInvalidQueryDirFiles, "lax", DefaultTolerateInvalidQueryDirFiles, "Tolerate invalid files in queryDir")

	flag.Parse()

	if service == "" {
		flag.Usage()
		log.Fatal("Error: URL to SQL Agent service required.")
	}

	if queriesFile == DefaultQueriesFile && queryDir != "" {
		queriesFile = ""
	}
	if queriesFile != "" && queryDir != "" {
		flag.Usage()
		log.Fatal("Error: You can specify either -queries or -queryDir")
	}

	var (
		err     error
		queries QueryList
		config  *Config
	)
	config = newConfig()
	if confFile != "" {
		config, err = loadConfig(confFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	if queryDir != "" {
		queries, err = loadQueriesInDir(queryDir, config, tolerateInvalidQueryDirFiles)
	} else {
		queries, err = loadQueryConfig(queriesFile, config)
	}
	if err != nil {
		log.Fatal(err)
	}

	if len(queries) == 0 {
		log.Fatal("No queries loaded!")
	}

	// Wait group of queries.
	wg := new(sync.WaitGroup)
	wg.Add(len(queries))

	// Shared context. Close the cxt.Done channel to stop the workers.
	ctx, cancel := context.WithCancel(context.Background())

	var w *Worker

	mux := http.NewServeMux()

	for _, q := range queries {
		// Create a new worker and start it in its own goroutine.
		// type key string
		// const wgKey key = "wg"
		w = NewWorker(context.WithValue(ctx, "wg", wg), q)
		go w.Start(service)
	}

	// Register the handler.
	mux.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("* Listening on %s...", addr)

	// Handles OS kill and interrupt.
	graceful.Run(addr, 5*time.Second, mux)

	log.Print("Canceling workers")
	cancel()
	log.Print("Waiting for workers to finish")
	wg.Wait()
	log.Println("All workers have finished, exiting!")
}
