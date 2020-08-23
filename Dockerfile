# Requires Docker v17.06 or later
FROM golang:1.15.0 as builder
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app
RUN go build -v .

FROM frolvlad/alpine-glibc:alpine-3.12_glibc-2.32
COPY --from=builder /go/src/app/prometheus-sql /usr/local/bin/prometheus-sql
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/prometheus-sql", "-host", "0.0.0.0"]
# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]