FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ARG VERSION=dev
ARG COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

COPY go.work ./
COPY services ./services

RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o /out/message ./services/message/cmd

FROM alpine:3.20

RUN apk --no-cache add ca-certificates
RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/message /usr/local/bin/message
COPY --from=builder /workspace/services/message/internal/config/config.yaml /app/config.yaml

WORKDIR /app
ENV CONFIG_PATH=/app/config.yaml

USER appuser

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -qO- http://localhost:8080/health || exit 1

EXPOSE 50054 8080

ENTRYPOINT ["/usr/local/bin/message"]
