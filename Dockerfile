# Requires Docker v17.06 or later
FROM golang:1.8.3 as builder
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY . /go/src/app
RUN go-wrapper download
RUN go-wrapper install

FROM frolvlad/alpine-glibc:alpine-3.5
COPY --from=builder /go/bin/app /usr/local/bin/prometheus-sql
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/prometheus-sql", "-host", "0.0.0.0"]
# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]