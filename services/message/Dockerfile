FROM golang:1.25-alpine AS builder

WORKDIR /workspace

COPY go.work ./
COPY services ./services

RUN go build -o /out/message ./services/message/cmd

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/message /usr/local/bin/message

USER appuser

EXPOSE 50054 8080

ENTRYPOINT ["/usr/local/bin/message"]
