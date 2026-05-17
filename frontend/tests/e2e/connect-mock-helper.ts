/**
 * ConnectRPC mock helper for Playwright E2E tests.
 *
 * Constructs proper binary Connect protocol responses so the
 * @connectrpc/connect-web client can deserialize them correctly.
 *
 * Wire format (Connect unary):
 *   Envelope: [type:1B][length:4B BE]   → 5 bytes
 *   Data:     serialized protobuf message
 *
 * Protobuf wire types:
 *   0 = varint   (int64, bool, uint64, sint64, etc.)
 *   2 = length-delimited  (string, bytes, embedded messages)
 *
 * Proto3 omits default/zero values on the wire.
 */

// ─── Protobuf wire encoding primitives ───────────────────────────────────────

/** Encode a signed/unsigned integer as a protobuf varint. */
function varint(value: number): number[] {
  const bytes: number[] = [];
  let v = value >>> 0;
  while (v > 0x7f) {
    bytes.push((v & 0x7f) | 0x80);
    v >>>= 7;
  }
  bytes.push(v & 0x7f);
  return bytes;
}

/** Wire tag = (field_number << 3) | wire_type */
function tag(fieldNumber: number, wireType: number): number[] {
  return varint((fieldNumber << 3) | wireType);
}

/** Length-delimited field: string, bytes, or embedded message (wire type 2). */
function lenDelimited(fieldNumber: number, valueBytes: number[]): number[] {
  return [...tag(fieldNumber, 2), ...varint(valueBytes.length), ...valueBytes];
}

/** Varint field: int64, bool, uint64, etc. (wire type 0). */
function varintField(fieldNumber: number, value: number): number[] {
  return [...tag(fieldNumber, 0), ...varint(value)];
}

/** Encode a UTF-8 string to byte array. */
function utf8(s: string): number[] {
  return [...new TextEncoder().encode(s)];
}

// ─── Message builders for known protobuf types ──────────────────────────────

/**
 * Build bytes for a Project message (aleph.v1.Project):
 *   string id = 1;
 *   string name = 2;
 *   int64 created_at = 3;
 */
function encodeProject(id: string, name: string): number[] {
  const idBytes = utf8(id);
  const nameBytes = utf8(name);
  return [
    ...lenDelimited(1, idBytes),    // id
    ...lenDelimited(2, nameBytes),  // name
    // created_at omitted (default 0 = omitted in proto3)
  ];
}

/**
 * Build bytes for an ApiKey message (aleph.v1.ApiKey):
 *   string id = 1;
 *   string label = 2;
 *   string key = 3;
 *   int64 created_at = 4;
 */
function encodeApiKey(id: string, label: string, key: string): number[] {
  return [
    ...lenDelimited(1, utf8(id)),    // id
    ...lenDelimited(2, utf8(label)), // label
    ...lenDelimited(3, utf8(key)),   // key
    // created_at omitted (default 0)
  ];
}

// ─── Connect envelope ────────────────────────────────────────────────────────

/**
 * Build a full Connect unary response body.
 *
 * Connect unary wire format (from @connectrpc/connect-web):
 *   Envelope: [flags:1B][length:4B BE]   → 5 bytes
 *   Data:     serialized protobuf message
 *
 * The flags byte uses the end-stream bit (bit 0 = 0x01).
 * When set, the client knows no more data follows.
 * Without it, the client will hang waiting for more envelopes.
 *
 * For unary responses, we send:
 *   1. Data envelope with end-stream flag set: [0x01][4-byte-length][message]
 *
 * @param messageBytes serialized protobuf message (empty = [])
 * @returns Uint8Array ready to send as response body
 */
function connectEnvelope(messageBytes: number[]): Uint8Array {
  const len = messageBytes.length;
  // Type byte: 0x01 = data + end-stream (unary response done)
  const header = [0x01, (len >> 24) & 0xff, (len >> 16) & 0xff, (len >> 8) & 0xff, len & 0xff];
  return new Uint8Array([...header, ...messageBytes]);
}

/**
 * Empty unary response (end-stream only, zero-length data).
 * Used for endpoints that return an empty protobuf message.
 * The type byte 0x01 signals both data and end-of-stream.
 */
const EMPTY = new Uint8Array([0x01, 0x00, 0x00, 0x00, 0x00]);

// ─── Top-level response builders for mocked endpoints ───────────────────────

/**
 * Build a response for `ProjectService.ListProjects`.
 * Returns `{ projects: [...] }`.
 */
export function mockListProjects(
  projects: { id: string; name: string }[] = [],
): { status: number; contentType: string; body: Uint8Array } {
  const fields: number[] = [];
  for (const p of projects) {
    const msg = encodeProject(p.id, p.name);
    // field 1 (repeated Project projects), wire type 2 (embedded message)
    fields.push(...lenDelimited(1, msg));
  }
  return { status: 200, contentType: 'application/connect+proto', body: connectEnvelope(fields) };
}

/**
 * Build a response for `ProjectService.CreateProject`.
 * Returns `{ project: { id, name } }`.
 */
export function mockCreateProject(
  id: string,
  name: string,
): { status: number; contentType: string; body: Uint8Array } {
  const projectBytes = encodeProject(id, name);
  const msg = lenDelimited(1, projectBytes); // field 1 (Project project)
  return { status: 200, contentType: 'application/connect+proto', body: connectEnvelope(msg) };
}

/**
 * Build a response for `AuthService.CreateApiKey`.
 * Returns `{ key: { id, label, key } }`.
 */
export function mockCreateApiKey(
  id: string,
  label: string,
  key: string,
): { status: number; contentType: string; body: Uint8Array } {
  const keyBytes = encodeApiKey(id, label, key);
  const msg = lenDelimited(1, keyBytes); // field 1 (ApiKey key)
  return { status: 200, contentType: 'application/connect+proto', body: connectEnvelope(msg) };
}

/**
 * Empty success response for any unary endpoint that returns a message
 * where all fields are optional or have defaults.
 * This covers: ListAgents, ListSkills, ListTools, ListTasks, ListAssets,
 * ListModels, ListApiKeys, ListNotificationChannels, GetOntology,
 * ListComponents, AnalyzeSentiment, Ping, etc.
 */
export function mockEmpty(): { status: number; contentType: string; body: Uint8Array } {
  return { status: 200, contentType: 'application/connect+proto', body: EMPTY };
}

// ─── Route helper ────────────────────────────────────────────────────────────

import type { Page, Route } from '@playwright/test';

type MockResponse = { status: number; contentType: string; body: Uint8Array };

/**
 * When Connect-ES sends requests with Content-Type: application/json,
 * decode the binary protobuf mock response and convert it to JSON response.
 * The Connect unary envelope still applies: [flags:1B][length:4BE][json-body].
 */
function jsonConnectBody(jsonObj: unknown): Uint8Array {
  const jsonStr = JSON.stringify(jsonObj);
  const jsonBytes = new TextEncoder().encode(jsonStr);
  const len = jsonBytes.length;
  const header = new Uint8Array([0x01, (len >> 24) & 0xff, (len >> 16) & 0xff, (len >> 8) & 0xff, len & 0xff]);
  const result = new Uint8Array(header.length + jsonBytes.length);
  result.set(header, 0);
  result.set(jsonBytes, header.length);
  return result;
}

/**
 * Convert a protobuf-binary mock to a JSON mock response.
 * Uses the path to infer the response shape.
 */
async function mockAsJson(route: Route, mock: MockResponse, path: string): Promise<void> {
  let jsonBody: unknown;

  if (path === '/aleph.v1.ProjectService/ListProjects') {
    jsonBody = { projects: [{ id: 'test', name: 'Test Project' }] };
  } else if (path === '/aleph.v1.ProjectService/CreateProject') {
    jsonBody = { project: { id: 'proj-1', name: 'TestWorkspace', createdAt: '0' } };
  } else if (path === '/aleph.v1.AuthService/CreateApiKey') {
    jsonBody = { key: { id: 'key-1', label: 'Admin Key (Wizard)', key: 'pw-test-api-key-00000', createdAt: '0' } };
  } else {
    jsonBody = {};
  }

  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: jsonConnectBody(jsonBody),
  });
}

/**
 * Register a route handler that intercepts all ConnectRPC calls and
 * returns the appropriate mock based on the request URL path.
 *
 * Fallback: returns an empty response (200 OK, no data) for unhandled endpoints.
 */
export async function setupApiMocks(page: Page): Promise<void> {
  const handlers: Record<string, () => MockResponse> = {
    // ProjectService
    '/aleph.v1.ProjectService/ListProjects': () =>
      mockListProjects([{ id: 'test', name: 'Test Project' }]),
    '/aleph.v1.ProjectService/CreateProject': () => mockCreateProject('proj-1', 'Test Project'),
    '/aleph.v1.ProjectService/DeleteProject': () => mockEmpty(),
    '/aleph.v1.ProjectService/GetOntology': () => mockEmpty(),
    '/aleph.v1.ProjectService/EmergeOntology': () => mockEmpty(),
    '/aleph.v1.ProjectService/SaveOntology': () => mockEmpty(),

    // AuthService
    '/aleph.v1.AuthService/CreateApiKey': () =>
      mockCreateApiKey('key-1', 'Admin Key (Wizard)', 'pw-test-api-key-00000'),
    '/aleph.v1.AuthService/ListApiKeys': () => mockEmpty(),
    '/aleph.v1.AuthService/DeleteApiKey': () => mockEmpty(),

    // AgentService
    '/aleph.v1.AgentService/ListAgents': () => mockEmpty(),
    '/aleph.v1.AgentService/CreateAgent': () => mockEmpty(),
    '/aleph.v1.AgentService/DeleteAgent': () => mockEmpty(),
    '/aleph.v1.AgentService/UpdateAgent': () => mockEmpty(),
    '/aleph.v1.AgentService/ListModels': () => mockEmpty(),

    // QueryService
    '/aleph.v1.QueryService/ExecuteQuery': () => mockEmpty(),
    '/aleph.v1.QueryService/GetChatHistory': () => mockEmpty(),
    '/aleph.v1.QueryService/GetDataStats': () => mockEmpty(),
    '/aleph.v1.QueryService/ConfirmAction': () => mockEmpty(),
    '/aleph.v1.QueryService/GlobalQuery': () => mockEmpty(),

    // IngestionService
    '/aleph.v1.IngestionService/ListTasks': () => mockEmpty(),
    '/aleph.v1.IngestionService/CreateTask': () => mockEmpty(),
    '/aleph.v1.IngestionService/DeleteTask': () => mockEmpty(),
    '/aleph.v1.IngestionService/RunTask': () => mockEmpty(),
    '/aleph.v1.IngestionService/GetTaskLogs': () => mockEmpty(),
    '/aleph.v1.IngestionService/GetProgress': () => mockEmpty(),

    // SkillService
    '/aleph.v1.SkillService/ListSkills': () => mockEmpty(),
    '/aleph.v1.SkillService/CreateSkill': () => mockEmpty(),
    '/aleph.v1.SkillService/DeleteSkill': () => mockEmpty(),

    // ToolService
    '/aleph.v1.ToolService/ListTools': () => mockEmpty(),
    '/aleph.v1.ToolService/CreateTool': () => mockEmpty(),
    '/aleph.v1.ToolService/DeleteTool': () => mockEmpty(),

    // LibraryService
    '/aleph.v1.LibraryService/ListAssets': () => mockEmpty(),
    '/aleph.v1.LibraryService/GetAssetContent': () => mockEmpty(),
    '/aleph.v1.LibraryService/DeleteAsset': () => mockEmpty(),
    '/aleph.v1.LibraryService/GeneratePdf': () => mockEmpty(),
    '/aleph.v1.LibraryService/UploadAsset': () => mockEmpty(),

    // NLP
    '/aleph.nlp.v1.NLPService/AnalyzeSentiment': () => mockEmpty(),

    // Registry
    '/aleph.registry.v1.RegistryService/ListComponents': () => mockEmpty(),

    // Notification
    '/aleph.v1.NotificationService/ListChannels': () => mockEmpty(),

    // Sandbox
    '/aleph.v1.SandboxService/RunSkill': () => mockEmpty(),
    '/aleph.v1.SandboxService/ExecuteTool': () => mockEmpty(),
  };

  // Intercept SSE (EventSource) endpoint — return a 200 with text/event-stream
  // content type and immediately close. This prevents infinite reconnection
  // loops from keeping the network busy.
  await page.route('/api/v1/events', async (route: Route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
      },
      body: ':\n\n', // SSE comment-only frame (keeps connection open without events)
    });
  });

  await page.route(/\/aleph\.(v1|nlp\.v1|registry\.v1|tool\.v1)\.\w+\/\w+/, async (route: Route) => {
    const url = route.request().url();
    const path = new URL(url).pathname;
    const contentType = route.request().headers()['content-type'] || '';

    const handler = handlers[path];
    if (handler) {
      const mock = handler();

      // The Connect-ES @connectrpc/connect-web library sends requests with
      // Content-Type: application/json in dev mode (or when content negotiation
      // prefers JSON). When the client sends JSON, we MUST respond with JSON
      // too — the client parses the binary envelope differently for proto vs JSON.
      //
      // For JSON mode, Connect unary envelope still uses [flags:1B][length:4BE],
      // but the body is JSON-encoded.
      if (contentType.includes('application/json')) {
        // Decode the protobuf binary response into JSON
        await mockAsJson(route, mock, path);
      } else {
        await route.fulfill({
          status: mock.status,
          contentType: mock.contentType,
          body: mock.body,
        });
      }
    } else {
      // Unknown endpoint — return empty 200
      await route.fulfill({
        status: 200,
        contentType: 'application/connect+proto',
        body: EMPTY,
      });
    }
  });
}

// ─── Zustand store helpers ───────────────────────────────────────────────────

/**
 * Set the zustand store state via page.evaluate.
 * Uses `useStore.setState()` which is accessible because the store is a module-level singleton.
 *
 * IMPORTANT: This requires the zustand store to be accessible via
 * the import system. In Vite dev mode, dynamic import works.
 */
export async function hydrateStore(page: Page, patch: Record<string, unknown>): Promise<void> {
  await page.waitForFunction(() => Boolean((window as Window & { __ALEPH_STORE__?: unknown }).__ALEPH_STORE__));
  await page.evaluate((data) => {
    const store = (window as Window & { __ALEPH_STORE__?: { setState: (patch: Record<string, unknown>) => void } }).__ALEPH_STORE__;
    store?.setState(data);
  }, patch);
}

/**
 * Call a zustand store action via page.evaluate.
 */
export async function callStoreAction(
  page: Page,
  actionName: string,
  ...args: unknown[]
): Promise<void> {
  await page.waitForFunction(() => Boolean((window as Window & { __ALEPH_STORE__?: unknown }).__ALEPH_STORE__));
  await page.evaluate(
    ({ name, params }) => {
      const store = (window as Window & {
        __ALEPH_STORE__?: {
          getState: () => Record<string, unknown>;
        };
      }).__ALEPH_STORE__;
      if (store) {
        const action = store.getState()[name];
        if (typeof action === 'function') {
          (action as (...a: unknown[]) => void)(...params);
        }
        return;
      }
    },
    { name: actionName, params: args },
  );
}
