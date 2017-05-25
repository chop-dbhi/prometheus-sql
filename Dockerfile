FROM frolvlad/alpine-glibc:alpine-3.5

ARG PROMETHEUS_SQL_VERSION=1.1.0
ARG BASE_URL=https://github.com/chop-dbhi/prometheus-sql/releases/download/$PROMETHEUS_SQL_VERSION
ARG OS_ARCH=linux-amd64
ARG ZIP_FILE=prometheus-sql-${OS_ARCH}.zip

ADD $BASE_URL/$ZIP_FILE /tmp/
RUN unzip /tmp/$ZIP_FILE -d /tmp \
  && cp /tmp/$OS_ARCH/* /usr/local/bin/ \
  && rm -rf /tmp/*

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/prometheus-sql", "-host", "0.0.0.0"]

# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]