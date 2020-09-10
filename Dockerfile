# Requires Docker v17.06 or later
FROM golang:1.15.2 as builder
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app
RUN go build -v .

FROM frolvlad/alpine-glibc:alpine-3.12_glibc-2.32

ENV PROMSQL_BIND_ADDRESS="0.0.0.0"
ENV PROMSQL_PORT="8080"

COPY --from=builder /go/src/app/prometheus-sql /usr/local/bin/prometheus-sql
COPY docker-entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/*

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]