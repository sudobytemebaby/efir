FROM golang:1.25-alpine AS builder

WORKDIR /workspace

ARG VERSION=dev
ARG COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

COPY go.work ./
COPY services ./services

RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o /out/room ./services/room/cmd

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin appuser

COPY --from=builder /out/room /usr/local/bin/room

USER appuser

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/room"]
