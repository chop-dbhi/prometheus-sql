FROM golang:1.7-onbuild

WORKDIR /

EXPOSE 8080

ENTRYPOINT ["app", "-host", "0.0.0.0"]

# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]
