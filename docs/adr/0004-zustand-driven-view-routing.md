# ADR-0004: Zustand-Driven View Routing (No React Router)

## Status

Accepted

## Context

Aleph Data OS frontend presents a terminal-inspired interface that behaves more like an operating system desktop than a website. The user interacts with a "terminal" and toggles between views: Terminal, Dashboard, Agents, Tools, DataSources, Components, Skills, Settings, Library, Explorer, Ontology, Oracle, and Health.

Traditional SPAs use URL-based routers (React Router, TanStack Router) where each view maps to a URL path. However, this paradigm does not fit Aleph's UX:

- Views are **workspace panels**, not navigable pages — users should not be able to bookmark "Settings" or share a URL to "Agent configuration"
- The terminal metaphor means the interface is modal (CMD vs INPUT mode) rather than browser-navigable
- URL changes would conflict with the OS-like illusion — the address bar is not part of the UX
- Some views (like SlideOver panels) overlay on top of the current view without changing the "active" view

## Decision

Use **Zustand store** for view state management instead of React Router or any URL-based router:

- **View enum**: `activeView` stored in Zustand's `navigationSlice` as a strict TypeScript enum
- **View switching**: Components read `activeView` from the store and render the corresponding view component
- **Code splitting**: Each view is loaded via `React.lazy()` with `Suspense` boundaries for bundle splitting
- **View transitions**: The `App.tsx` switch statement maps `activeView` values to lazy-loaded components
- **SlideOver panels**: Independent of `activeView` — they overlay the current view and do not change the navigation state
- **No URL dependencies**: The address bar is not used for routing state

This keeps the frontend as a single-page application with view switching controlled entirely by application state, not the URL.

## Consequences

### Positive
- No URL dependencies — cleaner state management with TypeScript control
- Tighter type safety for view transitions (enum values, not route strings)
- Simpler state management — view is just one field in the Zustand store
- No router library dependency — fewer bytes in the bundle
- View state is serializable and works with Zustand's devtools middleware
- SlideOver panels naturally coexist without routing conflicts

### Negative
- No deep linking — users cannot bookmark a specific view or share a URL to a particular screen
- No browser back/forward button for view navigation
- Debugging view state requires Zustand DevTools or store inspection
- New views must be manually registered in the View enum, the App.tsx switch, and navigation UI
- Potential for larger App.tsx as views are added (mitigated by lazy loading)

## Compliance

- New views registered in `frontend/src/types/index.ts` as members of the `View` enum
- View component added to the switch statement in `App.tsx` with `React.lazy` import
- No URL/router dependencies in new view components
- View components do not access browser history or location APIs
- State-based navigation only (via Zustand `navigationSlice.actions.setActiveView()`)

## Notes

- Zustand store structure: `navigationSlice` with `activeView: View` field and `setActiveView()` action
- Related ADRs: ADR-0005 (Zod for Frontend Validation + fromProto Mappers)
