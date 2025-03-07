FROM golang:1.23.7-alpine3.21 as go-builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tidb_writer /src/tidb/writer/tidb_workload.go


# ================================

FROM busybox:stable AS final

COPY --from=go-builder /src/tidb_writer /tidb_writer

RUN ls .

USER 0:0

ENTRYPOINT ["/tidb_writer"]