FROM golang:1.23.7-alpine3.21 AS go-builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o cassandra_writer /src/cassandra/writer/*.go


# ================================

FROM busybox:stable AS final

COPY --from=go-builder /src/cassandra_writer /cassandra_writer

RUN ls .

USER 0:0

ENTRYPOINT ["/cassandra_writer"]