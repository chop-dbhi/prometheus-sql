# Prometheus SQL

[![GoDoc](https://godoc.org/github.com/peakgames/prometheus-sql?status.svg)](https://godoc.org/github.com/peakgames/prometheus-sql)

Service that generates basic metrics for SQL result sets and exposing them as Prometheus metrics.

This service relies on the [SQL Agent](https://github.com/peakgames/sql-agent) service to execute and return the SQL result sets.

## Behavior

- Static configuration files are used to define the queries to monitor.
- Each query has a designated worker for execution.
- An interval is used to define how often to execute the query.
- Failed queries are automatically retried using a [backoff](https://en.wikipedia.org/wiki/Exponential_backoff) mechanism.
- Faceted metrics are supported.
- A single metric's different facets can be filled in from different data sources.

## Format

- Metric names are exposed in the format `query_result_<metric name>`.
- With faceted metrics, the name of the data column is determined by the `data-field` key in config, and all other columns (and column values) are exposed as labels.
- If the result set consists of a single row and column, the metric value is obvious and `data-field` is not needed.
- Label names under the same metric should be consistent.
- Each different query (query entry in config) for the same metric should lead to different label values.

## Build

```
go build
```

### Dependencies

Go dependencies are provided in the `vendor` dir. To update or restore dependencies, use [gvt](https://github.com/FiloSottile/gvt).


## Usage

Create a `queries.yml` file in the current directory that defines the set of named queries to monitor (see the [example file in this repository](./example-queries.yml)).

```
prometheus-sql
```

or for an alternate path, use the `-queries` option:

```
prometheus-sql -queries /path/to/queries.yml
```

or `-queryDir`, which will load all `*.yml` files in a directory:

```
prometheus-sql -queryDir /path/to/queries
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
