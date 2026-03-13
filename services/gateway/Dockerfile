FROM golang:1.25-alpine AS builder

WORKDIR /workspace

COPY go.work ./
COPY services ./services

RUN go build -o /out/gateway ./services/gateway/cmd

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/gateway /usr/local/bin/gateway

USER appuser

ENTRYPOINT ["/usr/local/bin/gateway"]
