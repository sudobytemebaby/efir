## ADR-005: NATS JetStream for Event-Driven Architecture

## Status

Accepted

## Context

The messenger requires asynchronous communication between services for events like user registration, room membership changes, and message creation. We need a message broker with at-least-once delivery guarantees.

## Decision

Use NATS JetStream as the event backbone with the following configuration:

### Streams

| Stream | Subjects | Retention | Storage | Max Age | Description |
|--------|----------|-----------|---------|---------|-------------|
| `AUTH` | `auth.>` | limits | file | 7d | Authentication events |
| `ROOM` | `room.>` | limits | file | 7d | Room events |
| `MESSAGE` | `message.>` | limits | file | 30d | Message events |

### Events

| Event | Publisher | Consumer | Payload |
|-------|-----------|----------|---------|
| `auth.user.registered` | Auth Service | User Service | `userId`, `email` |
| `room.membership.changed` | Room Service | WebSocket Connector | `roomId`, `userId`, `action`, `recipientIds` |
| `message.created` | Message Service | WebSocket Connector | `messageId`, `roomId`, `senderId`, `content`, `createdAt`, `recipientIds` |
| `room.updated` | Room Service | WebSocket Connector | `roomId`, `name`, `recipientIds` |

### Consumer Configuration

All consumers use:
- **Durable** subscriptions - survive restarts
- **AckExplicit** - manual acknowledgment
- **MaxDeliver: 5** - messages are discarded after 5 failed deliveries

When a message exceeds MaxDeliver attempts, it is logged as an error and discarded to prevent infinite retry loops on malformed messages.

### Payload Enrichment

The `recipientIds` field is populated by the publisher before sending:
- Room Service calls `GetRoomMembers()` and includes all member IDs
- WebSocket Connector receives complete payload - no additional API calls needed

## Rationale

- **JetStream over Core NATS**: Provides persistence and at-least-once delivery guarantees
- **Separate streams per domain**: Logical separation, independent retention policies
- **Durable consumers**: Survive service restarts without message loss
- **MaxDeliver: 5**: Prevents infinite retry loops on poison messages
- **Payload enrichment in publisher**: Reduces network hops, consumers receive complete data

## Alternatives Considered

- **NATS Core (non-persistent)**: Rejected - no delivery guarantee, messages lost on consumer restart
- **RabbitMQ**: More complex, heavier resource usage
- **Apache Kafka**: Overkill for single-developer project, complex运维

## Consequences

- Services communicate asynchronously via NATS events
- All consumers are responsible for creating their own durable subscriptions
- Failed message handling is automatic (discard after 5 retries)
- Event payloads are self-contained (recipientIds included)
