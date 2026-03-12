FROM golang:1.24-alpine AS builder

WORKDIR /workspace

COPY go.work ./
COPY services ./services

RUN go build -o /out/websocket ./services/websocket/cmd

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/websocket /usr/local/bin/websocket

USER appuser

ENTRYPOINT ["/usr/local/bin/websocket"]
