# Contributing to prometheus-sql

**First:** if you're unsure or afraid of _anything_, just ask or submit the issue or pull request anyways. You won't be yelled at for giving your best effort. The worst that can happen is that you'll be politely asked to change something. We appreciate any sort of contributions, and don't want a wall of rules to get in the way of that.

However, for those individuals who want a bit more guidance on the best way to contribute to the project, read on. This document will cover what we're looking for. By addressing all the points we're looking for, it raises the chances we can quickly merge or address your contributions.

## Issues

### Reporting an Issue

* Make sure you test against the latest released version. It is possible
  we already fixed the bug you're experiencing.

* Provide a reproducible test case. If a contributor can't reproduce an
  issue, then it dramatically lowers the chances it'll get fixed. And in
  some cases, the issue will eventually be closed.

* Respond promptly to any questions made by the maintainers of this project.
  Stale issues will be closed.

## Coding convention

* General guideline is to read some at the golang page [Effective Go](https://golang.org/doc/effective_go.html).
* Be sure to run [gofmt](https://golang.org/cmd/gofmt/) before you contribute.

## Opening an Pull Request

Please send a [GitHub Pull Request to prometheus-sql](https://github.com/chop-dbhi/prometheus-sql/pull/new/master) with a clear list of what you've done (read more about [pull requests](http://help.github.com/pull-requests/)).

## How To Build

### Build distribution packages

This section describes how you build prometheus-sql distribution packages.

1. Build prometheus-sql packages for several platforms:

    Using Make:
    ```bash
    make prepare dist
    ```

    Using Docker:
    ```bash
    docker run --rm -v ${PWD}:/go/bin -v ${PWD}:/usr/src/myapp -w /usr/src/myapp golang:1.8 bash -c 'apt-get update && apt-get -y install zip && make prepare dist'
    ```

2. All binaries produced are now located in directories under the `dist` directory.

### Build prometheus-sql Docker image locally

1. Go to the project root directory.

2. Build Docker image:

    Using Docker (require Docker v17.06 or later):
    ```bash
    docker build --rm --no-cache --tag dbhi/prometheus-sql:latest --file Dockerfile.multi .
    ```

3. Done!

