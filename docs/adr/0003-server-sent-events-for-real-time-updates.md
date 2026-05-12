# ADR-0003: Server-Sent Events for Real-Time Updates

## Status

Accepted

## Context

Aleph Data OS needs to push real-time events from backend to frontend: tool execution progress, status updates, orchestration state changes, prediction streaming, and health notifications. The frontend must receive these events without polling.

Options considered:

| Option | Direction | Native Browser | Reconnect | Complexity | Suitable For |
|--------|-----------|---------------|-----------|------------|--------------|
| WebSocket | Bidirectional | `new WebSocket()` | Manual | High | Chat, games, collaborative editing |
| SSE | Server→Client | `new EventSource()` | Auto | Low | Monitors, feeds, progress, logs |
| Polling | Client→Server | `fetch` + `setInterval` | N/A | Lowest | Low-frequency updates (< 5s) |
| WebSocket + RPC | Bidirectional | `new WebSocket()` | Manual | Highest | Full duplex |

Most real-time needs in Aleph are unidirectional (server → client). Tool execution progress, status updates, and notification streams do not require client-to-server streaming. For client-initiated requests, existing ConnectRPC unary calls are sufficient.

## Decision

Use **Server-Sent Events (SSE)** as the primary real-time push mechanism:

- **Backend**: SSE broker implemented at `internal/api/sse/` with subscription filtering and per-client event queues
- **Frontend**: `useSSE` React hook wrapping the native `EventSource` API, with auto-reconnect and event type routing
- **Event types**: Strongly typed event definitions shared between backend and frontend
- **Bidirectional streaming**: For cases where client → server streaming is needed, use ConnectRPC server-streaming RPC (never raw WebSocket)

The SSE endpoint is served at `/api/sse/` and uses standard HTTP/2 long-lived connections. The broker supports:
- Topic-based subscription filtering (e.g., subscribe to "tool:execution:*" events only)
- Per-client event queue with configurable buffer size
- Heartbeat keepalive to prevent proxy timeouts

## Consequences

### Positive
- Simple, HTTP-native protocol — no upgrade dance, works through all proxies
- Auto-reconnect via native `EventSource` — no custom reconnection logic
- Lower overhead than WebSocket for server→client-only push
- No additional port or protocol for firewalls/proxies
- Can be load-balanced like regular HTTP traffic

### Negative
- No server ← client streaming via SSE — client must use ConnectRPC unary calls to send data
- Browser limit of 6 concurrent SSE connections per domain (HTTP/1.1 only; HTTP/2 multiplexing effectively removes this)
- `EventSource` does not support custom headers — may need a polyfill for authenticated connections
- SSE is text-only — binary events require Base64 encoding
- Connection lifespan management (reconnect storms on network flapping)

## Compliance

- All new real-time push events go through the SSE broker at `internal/api/sse/`
- Frontend consumes SSE events via the `useSSE` hook in `frontend/src/hooks/`
- WebSocket is only added if bidirectional streaming is proven necessary via benchmarks and architecture review
- Event types are defined as typed constants in a shared file

## Notes

- `EventSource` API: https://developer.mozilla.org/en-US/docs/Web/API/EventSource
- Related ADRs: ADR-0002 (ConnectRPC over HTTP/2)
