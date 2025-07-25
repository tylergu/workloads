FROM golang:1.23.7-alpine3.21 AS go-builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o mongodb_writer /src/mongodb/writer_mongodb_1333/*.go


# ================================

FROM busybox:stable AS final

COPY --from=go-builder /src/mongodb_writer /mongodb_writer

RUN ls .

USER 0:0

ENTRYPOINT ["/mongodb_writer"]