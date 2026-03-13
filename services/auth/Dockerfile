FROM golang:1.25-alpine AS builder

WORKDIR /workspace

COPY go.work go.work.sum ./
COPY services ./services

RUN cd services/auth && go build -o /out/auth ./cmd

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/auth /usr/local/bin/auth

USER appuser

EXPOSE 50051 8080

ENTRYPOINT ["/usr/local/bin/auth"]
