FROM golang:1.23-alpine AS builder

WORKDIR /workspace

COPY go.work ./
COPY services ./services

RUN go build -o /out/room ./services/room/cmd

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/room /usr/local/bin/room

USER appuser

ENTRYPOINT ["/usr/local/bin/room"]
