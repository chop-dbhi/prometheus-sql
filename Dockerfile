FROM alpine:3.5

COPY ./dist/linux-amd64/prometheus-sql /

EXPOSE 8080

ENTRYPOINT ["/prometheus-sql", "-host", "0.0.0.0"]

# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]
