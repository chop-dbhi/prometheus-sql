# Prometheus SQL

[![GitHub release](https://img.shields.io/github/release/chop-dbhi/prometheus-sql.svg)](https://github.com/chop-dbhi/prometheus-sql)
[![Github Releases](https://img.shields.io/github/downloads/chop-dbhi/prometheus-sql/latest/total.svg)](https://github.com/chop-dbhi/prometheus-sql/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/dbhi/prometheus-sql.svg)](https://hub.docker.com/r/dbhi/prometheus-sql/)
[![GoDoc](https://godoc.org/github.com/chop-dbhi/prometheus-sql?status.svg)](https://godoc.org/github.com/chop-dbhi/prometheus-sql)

Service that generates basic metrics for SQL result sets and exposing them as Prometheus metrics.

This service relies on the [SQL Agent](https://github.com/chop-dbhi/sql-agent) service to execute and return the SQL result sets.

[Changelog](https://github.com/chop-dbhi/prometheus-sql/blob/master/CHANGELOG.md)

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

```shell
Usage of prometheus-sql:
  -config string
        Configuration file to define common data sources etc.
  -host string
        Host of the service. (0.0.0.0)
  -lax
        Tolerate invalid files in queryDir
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

In the repository there is an [example file](examples/example-queries.yml) that you can have a look at.

### Config file

The config file is optional and can defined some default values for queries and data sources which can be referenced by queries. The benefit of referencing a data source will be reduction of duplication of database connection information. See example config file [here](examples/working_example/config.yml) and [queries file](examples/working_example/queries.yml) which utilizes the config information.

### Run via console

Create a `queries.yml` file in the current directory and run the following:

```shell
prometheus-sql
```

or for an alternate path, use the -queries or the -queryDir option:

```shell
prometheus-sql -queries ${PWD}/queries.yml
```

### Run using Docker

Run the SQL agent service.

```bash
docker run -d --name sqlagent dbhi/sql-agent
```

Run this service. Mount the `queries.yml` file and link the SQL Agent service.

```shell
docker run -d \
  --name prometheus-sql \
  -p 8080:8080 \
  -v ${PWD}/queries.yml:/queries.yml \
  --link sqlagent:sqlagent \
  dbhi/prometheus-sql
```

If you want to separate database connection information etc you can do that by specifying data sources in separate file which you then can mount:

```shell
docker run -d \
  --name prometheus-sql \
  -p 8080:8080 \
  -v ${PWD}/queries.yml:/queries.yml \
  -v ${PWD}/prometheus-sql.yml:/prometheus-sql.yml \
  --link sqlagent:sqlagent \
  dbhi/prometheus-sql \
  -service http://sqlagent:5000 \
  -config prometheus-sql.yml
```

To view a plain text version of the metrics, open up the browser to the <http://localhost:8080/metrics> (or <http://192.168.59.103:8080/metrics> for boot2docker users).

### Run using a Docker Compose file

Alternately, use the `docker-compose.yml` file included in this repository. The `volumes` section be added for mounting the `queries.yml` file.

## Contributing

Read instructions [how to contribute](CONTRIBUTING.md) before you start.

## FAQ

**How do you I provide additional options to the database connection?**

Additional options are set in the `config.yml` file, specifically as additional key-value pairs in the [connection map](https://github.com/chop-dbhi/prometheus-sql/blob/master/examples/example-queries.yml#L14-L19). These are passed to the [SQL Agent service](https://github.com/chop-dbhi/sql-agent#connection-options) which construct a DSN string to establish the connection on the backend (alternately you can set the `dsn` key as the full string).

As an example, a common _gotcha_ when using Postgres in a development environment is to ignore SSL not being enabled on the server. This can be done by adding the `sslmode: disable` option in the connection map.

```yaml
    # ...
    connection:
        host: example.org
        port: 5432
        user: postgres
        password: s3cre7
        database: products
        sslmode: disable
```
