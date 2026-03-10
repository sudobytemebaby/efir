FROM golang:1.23-alpine AS builder

WORKDIR /workspace

COPY go.work ./
COPY services ./services

RUN go build -o /out/message ./services/message/cmd

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/message /usr/local/bin/message

USER appuser

ENTRYPOINT ["/usr/local/bin/message"]
