# ADR-0002: ConnectRPC over HTTP/2

## Status

Accepted

## Context

Aleph Data OS exposes an API surface that includes both unary request-response RPCs (query execution, configuration CRUD) and streaming RPCs (tool execution progress, real-time prediction streams). The API protocol selection needed to satisfy:

- **Browser support**: No proxy required for SPAs running on modern browsers
- **Streaming native**: Support for server-sent streaming without workarounds
- **TypeScript generation**: Strongly typed clients generated from a single definition
- **gRPC ecosystem compatibility**: Reuse of protobuf tooling, patterns, and conventions
- **Self-describing**: Schema-first definition with contract testing capability

Options evaluated:

| Protocol | Browser | Streaming | TS Gen | gRPC Compat | Proxy Needed |
|----------|---------|-----------|--------|-------------|--------------|
| gRPC | No | Yes | Yes | Yes | gRPC-web |
| REST/OpenAPI | Yes | SSE add-on | Manual | No | No |
| GraphQL | Yes | Subscriptions | Partial | No | No |
| ConnectRPC | Yes | Yes | Native | Full | No |

## Decision

Use **ConnectRPC** (via `bufbuild/connect-go` for Go and `@bufbuild/connect-query` for TypeScript) as the API protocol.

- All service definitions written as standard Protocol Buffer files in `api/proto/`
- Code generation via `buf generate` produces both Go server stubs and TypeScript client code
- ConnectRPC supports three protocols simultaneously from one definition: gRPC, gRPC-Web, and Connect (HTTP/JSON)
- The Connect protocol (HTTP/JSON) is used for browser communication — no gRPC-web proxy required
- Standard gRPC protocol used for backend-to-backend communication (NLP sidecar, MCP bridge)

Generated code is committed to the repository to ensure deterministic builds and easy code navigation.

## Consequences

### Positive
- Single proto definition serves all clients (Go backend, TypeScript frontend, Python sidecar)
- Native browser support via Connect protocol — no envoy/gRPC-web proxy needed
- First-class streaming RPC (server-streaming and bidirectional)
- Buf Schema Registry support available for service discoverability
- Strong TypeScript client generation guarantees frontend types match backend
- Can fall back to gRPC for inter-service calls within the backend

### Negative
- Additional codegen step in build pipeline (`buf generate`)
- Learning curve for team members unfamiliar with protobuf ecosystem
- Debugging binary protobuf on the wire is harder than plain JSON
- Proto field numbering discipline required — renumbering breaks wire compatibility
- Smaller community than REST/OpenAPI (though growing)

## Compliance

- All new service endpoints defined in `api/proto/` as `.proto` files
- Running `buf generate` to regenerate Go and TypeScript code before committing
- Generated code committed to repository
- No REST-only endpoints for new features without explicit architecture review
- All streaming RPCs use ConnectRPC server streaming (never raw WebSocket for new API endpoints)

## Notes

- ConnectRPC documentation: https://connectrpc.com/
- Buf CLI: https://buf.build/
- Related ADRs: ADR-0003 (Server-Sent Events for Real-Time Updates)
