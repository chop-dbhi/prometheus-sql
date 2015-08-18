# Prometheus SQL

[![GoDoc](https://godoc.org/github.com/chop-dbhi/prometheus-sql?status.svg)](https://godoc.org/github.com/chop-dbhi/prometheus-sql)

Service that generates basic metrics for SQL result sets and exposing them as Prometheus metrics.

This service relies on the [SQL Agent](https://github.com/chop-dbhi/sql-agent) service to execute and return the SQL result sets.

## Status

This is a prototype and may be merged into the SQL Agent repo directly.

## Behavior

- A static configuration file is used to define the queries to monitor.
- Each query has a designated worker for execution.
- An interval is used to define how often to execute the query.
- Failed queries are automatically retried using a [backoff](https://en.wikipedia.org/wiki/Exponential_backoff) mechanism.
- The result set of each query is held in memory for a subsequent comparison.
- Metrics are emitted for each query using a label (see [caveats](#caveats) below).
- The state of a query is exposed at `/state/<query>`

## Install

```
go get -u github.com/chop-dbhi/prometheus-sql
```

## Usage

Create a `queries.yml` file in the current directory that defines the set of named queries to monitor (see the [example file in this repository](./example-queries.yml)).

```
prometheus-sql
```

or for an alternate path, use the `-queries` option:

```
prometheus-sql -queries /path/to/queries.yml
```

## Docker

Run the SQL agent service.

```
docker run -d --name sqlagent dbhi/sql-agent
```


Run this service. Mount the queries.yml file and link the SQL Agent service.

```
docker run -d \
    -p 8080:8080 \
    -v /path/to/queries.yml:/queries.yml \
    --link sqlagent:sqlagent \
    dbhi/prometheus-sql
```

To view a plain text version of the metrics, open up the browser to the http://localhost:8080 (or http://192.168.59.103:8080/metrics for boot2docker users).


### Compose

Alternately, use the `docker-compose.yml` file included in this repository. The `volumes` section be added for mounting the `queries.yml` file.


## Caveats

Currently, the Prometheus client library does not support creating a separate registry (of metrics) for each query, so metrics are differentiated using a `query:<name>` label. The side effect to this is that there is a single endpoint for all queries rather than one endpoint per query. See [prometheus/client_golang#46](https://github.com/prometheus/client_golang/issues/46) for details.
