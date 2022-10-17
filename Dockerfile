# Requires Docker v17.06 or later
FROM golang:1.18.3 as builder
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app
RUN go build -v .

FROM frolvlad/alpine-glibc:alpine-3.16_glibc-2.35

COPY --from=builder /go/src/app/prometheus-sql /usr/local/bin/prometheus-sql

RUN chmod +x /usr/local/bin/*

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/prometheus-sql"]
# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]