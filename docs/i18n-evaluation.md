# i18n Evaluation — Aleph-v2

## Current State
- All UI text hardcoded in Italian
- Only `SetupWizard` has a toggle for EN/IT
- No i18n framework in use

## Options

### Option A: react-i18next
- **Pros**: Mature, lazy loading, TypeScript support, ICU message format
- **Cons**: ~8KB bundle overhead
- **Estimate**: 3-4 days for full migration

### Option B: next-intl
- **Pros**: Lightweight, RTL support, good for SSG
- **Cons**: Primarily designed for Next.js
- **Estimate**: 2-3 days

### Option C: Custom solution
- **Pros**: Zero deps, minimal bundle
- **Cons**: Must build pluralization, interpolation, lazy loading
- **Estimate**: 4-5 days

## Recommendation
**Option A** (react-i18next) for now — mature, well-supported, works with any React setup.
Deferred until post-W6 (Autocoscienza).
