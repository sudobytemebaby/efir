#!/bin/bash
set -euo pipefail

NATS_URL="${NATS_URL:-http://nats:4222}"
NATS_USER="${NATS_USER:-nats}"
NATS_PASS="${NATS_PASSWORD:-nats}"

create_stream() {
    local stream_name="$1"
    local subjects="$2"

    echo "Creating stream $stream_name with subjects: $subjects"

    curl -s -X PUT "${NATS_URL}/jsz/streams" \
        -u "${NATS_USER}:${NATS_PASS}" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"${stream_name}\",
            \"subjects\": [\"${subjects}\"],
            \"retention\": \"limits\",
            \"storage\": \"file\",
            \"max_bytes\": 1073741824,
            \"max_age\": 604800
        }" || echo "Stream $stream_name may already exist"
}

main() {
    echo "Waiting for NATS to be ready..."
    timeout 30 bash -c 'until curl -s -u ${NATS_USER}:${NATS_PASS} ${NATS_URL}/healthz > /dev/null 2>&1; do sleep 1; done'

    echo "Creating NATS JetStream streams..."

    create_stream "AUTH" "auth.>"
    create_stream "ROOM" "room.>"
    create_stream "MESSAGE" "message.>"

    echo "NATS JetStream streams created successfully"
}

main "$@"
