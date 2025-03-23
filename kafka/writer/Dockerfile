FROM golang:1.23-bullseye AS go-builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o kafka_writer /src/kafka/writer/*.go

USER 0:0

ENTRYPOINT ["/src/kafka_writer"]