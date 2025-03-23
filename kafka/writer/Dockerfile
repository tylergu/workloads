FROM golang:1.23.7-alpine3.21 AS go-builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o kafka_writer /src/kafka/writer/*.go


# ================================

FROM busybox:stable AS final

COPY --from=go-builder /src/kafka_writer /kafka_writer

RUN ls .

USER 0:0

ENTRYPOINT ["/kafka_writer"]