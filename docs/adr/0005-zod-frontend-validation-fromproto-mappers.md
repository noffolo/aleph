# ADR-0005: Zod for Frontend Validation + fromProto Mappers

## Status

Accepted

## Context

Aleph's frontend receives data from the backend in Protocol Buffer format via ConnectRPC. The protobuf-generated TypeScript types carry the schema of the wire format, but three distinct validation needs remain:

1. **Stricter-than-proto validation**: Protobuf types may allow `null`, `undefined`, or `""` where the frontend needs non-null guaranteed values. String length limits, number ranges, and enum validation must be enforced client-side before rendering.

2. **Form validation**: User input forms (agent creation, data source configuration) need real-time validation before submission. Protobuf types are not designed for form validation — they describe the final message shape, not intermediate input states.

3. **Transformation layer**: The protobuf wire format does not always match the ideal frontend model. Dates may arrive as timestamps needing conversion, nested messages may need flattening, and fields may need renaming for ergonomic component access.

Without a dedicated frontend validation layer, these concerns leak into UI components — mixing display logic with validation logic.

## Decision

Use **Zod** as the frontend validation library with a clear two-layer architecture:

### Layer 1: fromProto Mappers
Dedicated functions in `frontend/src/schemas/` whose sole responsibility is converting protobuf response types to Zod-validated frontend types. These are named `fromProto*` (e.g., `fromProtoToolRecord`, `fromProtoAgentConfig`) and:

- Accept the protobuf-generated type as input
- Call `zodSchema.parse(protoData)` to apply validation
- Return the validated frontend type
- Throw descriptive errors on validation failure

### Layer 2: Zod Schemas
Zod schemas define the "source of truth" for frontend data shapes:

- **Response schemas**: Define what the frontend expects to receive from the API, co-located with fromProto mappers
- **Form schemas**: Define form input validation rules, used by form libraries (react-hook-form via `@hookform/resolvers/zod`)
- **Derived schemas**: Compute derived values (e.g., `z.object().transform()` for date formatting)

### Rules
- No raw protobuf type usage in UI components — always go through fromProto mappers
- Form submits validated via Zod schema before calling API
- Error messages from Zod's `format()` used for inline field validation display

## Consequences

### Positive
- Type-safe validation chain: proto → Zod → UI components
- fromProto mappers are single-responsibility functions, easily testable in isolation
- Zod's `parse()` throws descriptive, stack-traced errors on data mismatch
- Schemas co-located with forms they validate — easy to find and modify
- runtime validation catches API contract violations early in development
- `z.infer<typeof schema>` provides full TypeScript type inference

### Negative
- Boilerplate for each new response type — must create schema + mapper for every API response
- fromProto mappers need maintenance when proto fields change (dual maintenance)
- Schema definition duplication risk (proto schema + Zod schema describing the same shape)
- Zod parse overhead on every response (negligible for typical payload sizes)
- Additional bundle size (~12KB gzipped for zod)

## Compliance

- All form inputs validated with Zod before submit (using `@hookform/resolvers/zod`)
- All protobuf-to-frontend conversions go through fromProto mapper functions in `frontend/src/schemas/`
- No raw protobuf types imported directly in UI component files
- Every protobuf `*Service` response type has a corresponding Zod schema and fromProto mapper
- Tests exist for mapper parse failures (malformed proto data)

## Notes

- Zod documentation: https://zod.dev/
- react-hook-form + Zod integration: https://github.com/react-hook-form/resolvers
- Related ADRs: ADR-0002 (ConnectRPC over HTTP/2), ADR-0004 (Zustand-Driven View Routing)
