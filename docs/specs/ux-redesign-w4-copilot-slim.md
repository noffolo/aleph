# W4 — Copilot Slim: UX Redesign Spec

> **Wave:** W4 (Copilot Slim)
> **Status:** Draft
> **Target:** `CopilotView.tsx` (282→~120 lines), `copilotSlice.ts` (10→6 state fields)
> **Risk:** Medium — affects chat UX, store shape, and scene routing

---

## 1. Current Problems

### 1.1 Scope Creep — Copilot Is a Command Center, Not a Chat

`CopilotView.tsx` (282 lines) has accumulated responsibilities that belong elsewhere:

| Feature | Lines | Problem |
|---------|-------|---------|
| Chat messages + input + streaming | ~80 | Core — belongs here |
| Split view / message detail panel | ~50 | View-mode logic, not chat |
| Inline command dropdown (5 hardcoded cmds) | ~60 | Duplicates `CommandPalette`'s 16 slash commands |
| `ChatSearchBar` (inline search) | ~15 | Duplicates `CommandPalette`'s unified search |
| Confirm dialog (approve/reject) | ~8 | Should be shared Toast/notification |
| SSE status indicator | ~20 | Useful but duplicated (header + footer) |
| Bookmark / filter-to-bookmarks | state only | Never used in UI, dead code |

### 1.2 Dual Command Systems

- **CopilotView** renders 5 hardcoded commands (`help`, `clear`, `settings`, `agents`, `tools`) in a custom dropdown triggered by `/` prefix.
- **CommandPalette** has 16 `SLASH_COMMANDS` (imported from `slashCommands.ts`), Tab autocomplete, and unified search across objects/projects/commands.

These overlap: `/tools` exists in both but the Copilot version is a static list without the `executeCommand()` dispatch logic.

### 1.3 Full-Store Subscription → Re-render Cascade

```typescript
const { splitView, setSplitView, chatSearchQuery, setChatSearchQuery } = useStore()
```

`useStore()` subscribes to **every** state change in all 6 slices. Every keystroke, health check, or workspace update triggers a re-render of the entire `CopilotView` tree (~280 lines). The component is wrapped in `React.memo` but the full-store subscription bypasses it.

### 1.4 Dead / Orphaned State

| copilotSlice Field | Used in UI? | Notes |
|--------------------|-------------|-------|
| `bookmarkedIds` | ❌ No | `Set<number>` with `toggleBookmark` — no visible UI, no filter |
| `onlyBookmarks` | ❌ No | Boolean switch — no visible UI |
| `chatSearchQuery` | ✅ Yes | Via `ChatSearchBar` — but duplicates CommandPalette |
| `splitView` | ✅ Yes | Collides with `showInlinePanel` in navigationSlice |

### 1.5 `splitView` vs `showInlinePanel` Collision

- `copilotSlice.splitView` — splits chat pane to show message detail on the right.
- `navigationSlice.showInlinePanel` — shows a bottom panel for inline rendering.

Both control "show a secondary view near the chat" with zero coordination. After W3's SlideOver migration, neither should live in its current form — scene routing via `slideOverContent` replaces both.

---

## 2. Simplified Architecture

### 2.1 Principle: Copilot is Pure Chat

After W4, `CopilotView` does exactly one thing: **send messages, display messages, stream responses**. Everything else belongs to:

| Responsibility | New Owner |
|---------------|-----------|
| Search messages | `CommandPalette` (Cmd+K already unified) |
| Slash commands | `CommandPalette` via `SLASH_COMMANDS` |
| Split view / message detail | `SlideOver` with scene context |
| Inline renderer panel | `navigationSlice.showInlinePanel` (already exists) |
| Confirm dialogs | Shared `ToastContainer` + notification system |
| Bookmarking | Removed (dead code) |

### 2.2 Target Component Structure

```
CopilotView (~120 lines)
├── Header bar: agent selector + SSE status (condensed) + clear/stop
├── TerminalOutput: message list with auto-scroll
├── Streaming cursor (inline, no separate component)
└── TerminalPrompt: input bar (local state, not store)
```

### 2.3 Target Store Shape

**New `copilotSlice` (~6 fields):**

```typescript
interface CopilotSlice {
  messages: ChatMessage[]        // was `chat`
  isStreaming: boolean
  streamingMessage: string       // partial SSE content
  streamingToolCalls: ToolCall[]
  abortController: AbortController | null
  selectedAgent: string
}
```

**Removed from `copilotSlice`:**

| Field | Disposition |
|-------|-------------|
| `input` | Local `useState` in `CopilotView` — no reason for global store |
| `splitView` | Removed. Scene routing via `slideOverContent` |
| `setSplitView` | Removed |
| `bookmarkedIds` | Removed — dead code, no UI uses it |
| `toggleBookmark` | Removed |
| `chatSearchQuery` | Removed — use CommandPalette search |
| `setChatSearchQuery` | Removed |
| `onlyBookmarks` | Removed — dead code |
| `setOnlyBookmarks` | Removed |
| `pendingConfirmation` | Migrate to shared notification queue |
| `setPendingConfirmation` | Removed from slice |
| `resetCopilot` | Slimmed to reset only the 6 remaining fields |

---

## 3. Feature Consolidation Plan

### 3.1 Search: Inline `ChatSearchBar` → CommandPalette

**Current:** `CopilotView` renders `<ChatSearchBar>` which filters messages client-side by content/toolCall substring.

**Target:** Remove `ChatSearchBar` import and `chatSearchQuery` state. Users search chat via `Cmd+K` → CommandPalette, which already has:
- Filterable slash commands (16 items from `SLASH_COMMANDS`)
- Object/project search
- Tab autocomplete

**Gap:** CommandPalette does not currently search chat messages. This must be added:
- Add a `searchMessages` function to `copilotSlice.selectors`
- CommandPalette gains a new section "Messages" when the user types a natural query (no `/` prefix)
- Results link back to the chat and highlight matching text
- Keyboard shortcut `Cmd+Shift+F` opens CommandPalette pre-filtered to messages

**Migration steps:**
1. Remove `<ChatSearchBar>` from `CopilotView.tsx`
2. Remove `chatSearchQuery`, `setChatSearchQuery` from `copilotSlice`
3. Add message search to `CommandPalette.tsx` (new section for non-slash queries)
4. Add `Cmd+Shift+F` binding to open CommandPalette in message-search mode

### 3.2 Commands: Dual System → Single CommandPalette

**Current:** Two command systems:
1. CopilotView: 5 hardcoded commands, `/` prefix, inline dropdown (lines 55-66, 219-241)
2. CommandPalette: 16 SLASH_COMMANDS, `Cmd+K`, Tab autocomplete, 3 result sections

**Target:** Remove CopilotView's inline command dropdown entirely. All commands flow through `CommandPalette`.

**Migration steps:**
1. Delete lines 55-66 (hardcoded `commands` array) and lines 219-241 (command dropdown JSX) from `CopilotView.tsx`
2. Delete `showCommands` and `commandInput` local state from CopilotView
3. Remove the `/`-prefix detection `useEffect` (lines 46-53)
4. Verify all 5 hardcoded commands exist in `SLASH_COMMANDS` (add any missing ones)
5. CommandPalette keyboard shortcut `Cmd+K` is the single entry point

**Tab autocomplete** continues to work in CommandPalette via `getTabCompletion()` — no changes needed.

### 3.3 Split View → Scene Routing

**Current:** `splitView` boolean toggles the right panel showing message details. Directly competes with `showInlinePanel`.

**Target:** Remove `splitView` from copilotSlice. The message-detail panel becomes a `SlideOver` that opens with `setSlideOverContent({ type: 'message-detail', data: { message, index } })`.

**Scene context:** The `slideOverContent.type` already supports `'detail'` — this can host message details. The `InlineRenderer` (bottom panel) continues using `navigationSlice.showInlinePanel` since it's a different position (bottom vs. side).

**Migration steps:**
1. Remove `splitView`, `setSplitView` from `copilotSlice`
2. Replace split-view toggle button (lines 120-127) with a "message detail" action in the message context
3. When user selects a message (e.g., right-click or info icon), dispatch `setSlideOverContent({ type: 'detail', title: 'Message Detail', data: message })`
4. Remove the `splitView && (...)` JSX block (lines 187-212)

### 3.4 Confirm Dialog → Shared Notification

**Current:** Lines 244-250 render inline approve/reject buttons when `chat.some(m => m.requiresConfirmation)`.

**Target:** Remove inline confirmation. Use a shared notification queue (e.g., `uiSlice.notifications`) that renders through `ToastContainer`.

**Migration steps:**
1. Remove lines 244-250 from `CopilotView.tsx`
2. Move `pendingConfirmation` from `copilotSlice` to a `notifications[]` array in `uiSlice`
3. ToastContainer reads `notifications` and renders confirm prompts as persistent toasts with approve/reject buttons
4. Remove `onConfirmAction` from props forwarding

### 3.5 Bookmarks → Removed

`bookmarkedIds` and `onlyBookmarks` in copilotSlice have zero UI rendering. The `toggleBookmark` action exists but no component calls it.

**Migration:** Remove the 4 fields/methods unconditionally. If bookmarking is needed later, implement it as a `Set<string>` on message `id` in a dedicated slice.

### 3.6 SSE Status: Deduplicate

**Current:** CopilotView renders SSE status dots/labels twice:
1. Header bar (lines 140-154): colored dot + reconnect count
2. Footer bar (lines 252-269): full "SSE: connected" label + dot

**Target:** Single SSE status in the footer (more natural position — near the input bar).

### 3.7 `input` → Local State

**Current:** `copilotSlice.input` and `copilotSlice.setInput` — re-renders every subscriber on every keystroke.

**Target:** Local `useState('')` inside `CopilotView`. Only `onSend()` dispatches the message to the store. This eliminates the keystroke re-render cascade.

---

## 4. Removed Features & Migration Table

| Feature | Removed From | Moved To | Lines Saved |
|---------|-------------|----------|-------------|
| `ChatSearchBar` | CopilotView | CommandPalette (new message section) | ~15 |
| Inline command dropdown | CopilotView | CommandPalette (already exists) | ~60 |
| `splitView` + message detail | copilotSlice | SlideOver (`slideOverContent.type='detail'`) | ~50 |
| Confirm dialog JSX | CopilotView | ToastContainer (`uiSlice.notifications`) | ~8 |
| `bookmarkedIds` / `onlyBookmarks` | copilotSlice | **Removed** (dead code) | N/A |
| `input` in store | copilotSlice | `useState` in CopilotView | N/A |
| `pendingConfirmation` in slice | copilotSlice | `uiSlice.notifications[]` | N/A |
| Duplicate SSE in header | CopilotView | Footer only | ~15 |
| `showCommands` / `commandInput` state | CopilotView | **Removed** | ~8 |
| `/`-prefix detection `useEffect` | CopilotView | **Removed** | ~8 |

**Total line reduction:** ~282 → ~120 lines (**~57% slimmer**)

### 4.1 Backward Compatibility

- **Store reads:** In a transitional release, removed fields are marked `@deprecated` with a no-op getter that logs a warning. Consumers (tests, legacy components) continue to work. Removal in next release.
- **`resetCopilot`:** Slimmed to only reset the 6 remaining fields. Legacy `resetCopilot` callers still work since removed fields are no longer in the type.
- **`splitView` readers:** Any component reading `splitView` gets `false` during deprecation window. Components using `showInlinePanel` from navigationSlice continue unchanged.

---

## 5. Selector Migration

### 5.1 Problem

```typescript
// Current — full-store subscription
const { chat, isStreaming, input } = useStore()
```

### 5.2 Solution

Create granular selectors on the copilot slice:

```typescript
// New — scoped selectors
const messages = useStore(s => s.messages)
const isStreaming = useStore(s => s.isStreaming)
```

Every component that touches copilot state migrates from `useStore()` destructuring to `useStore(selector)` with a single-field selector. This eliminates unnecessary re-renders.

**Affected files:** `CopilotView.tsx`, `TerminalPrompt.tsx` (if it reads store), `InlineRenderer.tsx`, any hook that reads `copilotSlice`.

**Priority order:**
1. `CopilotView.tsx` — P0, the main re-render culprit
2. `InlineRenderer.tsx` — P1, renders alongside Copilot
3. Any tests using `useStore()` — P2

---

## 6. Execution Plan

### Phase 1 — Slice Surgery (copilotSlice.ts)
1. Remove: `input`, `setInput`, `splitView`, `setSplitView`, `bookmarkedIds`, `toggleBookmark`, `chatSearchQuery`, `setChatSearchQuery`, `onlyBookmarks`, `setOnlyBookmarks`
2. Deprecate: `pendingConfirmation`, `setPendingConfirmation` (add deprecation JSDoc + warn-on-read wrapper)
3. Add: `streamingMessage: string`, `streamingToolCalls: ToolCall[]`
4. Update `resetCopilot` to reflect new field set
5. Rename `chat` → `messages` (or keep `chat` with alias — lean toward rename for clarity)
6. Verify TypeScript compilation: `npx tsc --noEmit`

### Phase 2 — CopilotView Rewrite
1. Remove: `ChatSearchBar` import + JSX, inline command list + dropdown, `splitView` panel, confirm dialog, duplicate SSE header
2. Convert: `input` / `setInput` from store → local `useState`
3. Refactor: `useStore()` → granular `useStore(s => s.field)` selectors
4. Replace: splitView button → no view-toggle (scene routing handles it)
5. Condense: SSE status to footer only (remove header copy)
6. Verify: `npx tsc --noEmit`, `npx vite build`

### Phase 3 — CommandPalette Enhancement
1. Add: message search section to `CommandPalette.tsx` (for non-slash natural queries)
2. Add: `Cmd+Shift+F` keyboard shortcut bound to CommandPalette pre-filtered for messages
3. Verify: `executeCommand()` path for all previously hardcoded commands

### Phase 4 — Confirm Dialog Migration
1. Add: `notifications` to `uiSlice` (array of `{ id, type, message, actions }`)
2. Create: confirm-toast variant in `ToastContainer`
3. Remove: inline approve/reject from `CopilotView`
4. Wire: `pendingConfirmation` → `notifications` bridge during deprecation window

### Phase 5 — Cleanup & Test
1. Run: `go build ./...` (backend — verify no RPC shape changes)
2. Run: `npx tsc --noEmit` (frontend — verify type safety)
3. Run: `npx vite build` (verify bundle size reduction)
4. Run: `npx vitest run` (verify no test regressions)
5. Run: `gitnexus_detect_changes()` — confirm only expected files changed

---

## Appendix: State Shape Diff

### Before (copilotSlice, 10 fields)

```typescript
chat: ChatMessage[]
input: string
isStreaming: boolean
streamAbortController: AbortController | null
pendingConfirmation: PendingConfirmation | null
selectedAgent: string
splitView: boolean
bookmarkedIds: Set<number>
chatSearchQuery: string
onlyBookmarks: boolean
```

### After (copilotSlice, 6 fields + 2 new)

```typescript
messages: ChatMessage[]
isStreaming: boolean
streamingMessage: string          // NEW — partial SSE content
streamingToolCalls: ToolCall[]    // NEW — streaming tool calls
abortController: AbortController | null  // renamed from streamAbortController
selectedAgent: string
```

### Before (CopilotView local state, 4 fields)

```typescript
selectedMsgIndex: number | null
isAtBottom: boolean
showCommands: boolean
commandInput: string
```

### After (CopilotView local state, 2 fields)

```typescript
input: string                     // moved from store
isAtBottom: boolean
```
