FROM alpine:edge AS build
RUN apk add --no-cache --update go gcc g++
WORKDIR /app
ENV GOPATH /app
COPY . /app/src

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o kafka_writer /app/src/kafka/writer/*.go


# ================================

FROM busybox:stable AS final

COPY --from=build /src/kafka_writer /kafka_writer

RUN ls .

USER 0:0

ENTRYPOINT ["/kafka_writer"]