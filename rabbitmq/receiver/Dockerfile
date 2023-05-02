FROM golang:1.16-alpine3.14 as go-builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o receiver /src/rabbitmq/receiver/receiver.go


# ================================

FROM busybox:stable AS final

COPY --from=go-builder /src/receiver /receiver

RUN ls .

USER 0:0

ENTRYPOINT ["/receiver"]