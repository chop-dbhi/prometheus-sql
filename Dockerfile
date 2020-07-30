# Requires Docker v17.06 or later
FROM golang:1.14.6 as builder
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app
RUN go get -u github.com/golang/dep/cmd/dep
RUN go install github.com/golang/dep/cmd/dep
RUN dep ensure
RUN go install

FROM frolvlad/alpine-glibc:alpine-3.12_glibc-2.31
COPY --from=builder /go/bin/app /usr/local/bin/prometheus-sql
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/prometheus-sql", "-host", "0.0.0.0"]
# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]
