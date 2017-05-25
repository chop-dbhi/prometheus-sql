# Prometheus SQL

[![GitHub release](https://img.shields.io/github/release/chop-dbhi/prometheus-sql.svg)](https://github.com/chop-dbhi/prometheus-sql)
[![Docker Pulls](https://img.shields.io/docker/pulls/dbhi/prometheus-sql.svg)](https://hub.docker.com/r/dbhi/prometheus-sql/)
[![Docker Build Status](https://img.shields.io/docker/build/dbhi/prometheus-sql.svg)](https://hub.docker.com/r/dbhi/prometheus-sql/builds/)
[![GoDoc](https://godoc.org/github.com/chop-dbhi/prometheus-sql?status.svg)](https://godoc.org/github.com/chop-dbhi/prometheus-sql)

Service that generates basic metrics for SQL result sets and exposing them as Prometheus metrics.

This service relies on the [SQL Agent](https://github.com/chop-dbhi/sql-agent) service to execute and return the SQL result sets.

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

## Usage

```bash
Usage of prometheus-sql:
  -host string
        Host of the service.
  -port int
        Port of the service. (default 8080)
  -queries string
        Path to file containing queries. (default "queries.yml")
  -queryDir string
        Path to directory containing queries.
  -service string
        Query of SQL agent service.
```

### Queries file

A queries file is required for the application to know which data source to query and which queries that shall be monitored.

In the repository there is an [example file](example-queries.yml) that you can have a look at.

### Run via console

Create a `queries.yml` file in the current directory and run the following:

```bash
prometheus-sql
```

or for an alternate path, use the -queries or the -queryDir option:

```bash
prometheus-sql -queries /path/to/queries.yml
```

### Run using Docker

Run the SQL agent service.

```bash
docker run -d --name sqlagent dbhi/sql-agent
```

Run this service. Mount the `queries.yml` file and link the SQL Agent service.

```bash
docker run -d \
    --name prometheus-sql \
    -p 8080:8080 \
    -v /path/to/queries.yml:/queries.yml \
    --link sqlagent:sqlagent \
    dbhi/prometheus-sql
```

To view a plain text version of the metrics, open up the browser to the <http://localhost:8080/metrics> (or <http://192.168.59.103:8080/metrics> for boot2docker users).


### Run using a Docker Compose file

Alternately, use the `docker-compose.yml` file included in this repository. The `volumes` section be added for mounting the `queries.yml` file.


## Contributing

Read instructions [how to contribute](CONTRIBUTING.md) before you start.
