FROM frolvlad/alpine-glibc:alpine-3.5

ARG PROMETHEUS_SQL_VERSION=1.2.0
ARG PKG_FILE=prometheus-sql-linux-amd64.tar.gz
ARG PKG_URL=https://github.com/chop-dbhi/prometheus-sql/releases/download/$PROMETHEUS_SQL_VERSION/$PKG_FILE

RUN apk update
RUN apk add curl

# Workaround to publish Github release before curl fetches it.
RUN sleep 60

RUN mkdir -p /opt/prometheus-sql/bin \
    && curl --retry 5 --retry-delay 15 --retry-max-time 120 -SL $PKG_URL \
    | tar -xzC /opt/prometheus-sql/bin \
    && ln -s /opt/prometheus-sql/bin/linux-amd64/prometheus-sql /usr/local/bin/

RUN apk del curl
RUN rm /var/cache/apk/*

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/prometheus-sql", "-host", "0.0.0.0"]

# Default command assumes the SQL agent is linked.
CMD ["-service", "http://sqlagent:5000"]