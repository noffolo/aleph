# CSS Purge-Safe Audit Report (frontend/)

## Overview
This audit identifies dynamic Tailwind CSS class constructions in the `frontend/src/` directory that may be purged by CSS optimizers (e.g., PurgeCSS, Tailwind's JIT engine) because they are not present as full, static strings in the source code.

## Summary
- **Total Dynamic Patterns Found:** 21
- **Critical Risks:** 1 (Computed colors from variables)
- **Moderate Risks:** 18 (Conditional template literals)
- **Low Risks:** 2 (Non-Tailwind dynamic strings)

---

## Findings

### 🔴 Critical Risk: Computed Class Names
These patterns construct classes by combining a prefix with a variable. Tailwind's JIT engine **cannot** detect these.

| File | Line | Pattern | Risk | Recommendation |
| :--- | :--- | :--- | :--- | :--- |
| `src/components/GenericCommandPalette.tsx` | 88 | \`bg-$\{item.colorClass || 'primary'}/10\` | **High** - Classes generated at runtime. | Use a lookup table: `const colorMap = { primary: 'bg-primary/10', ... }` |
| `src/components/GenericCommandPalette.tsx` | 90 | \`text-$\{item.colorClass || 'primary'\}\` | **High** - Classes generated at runtime. | Use a lookup table. |
| `src/components/GenericCommandPalette.tsx` | 92 | \`text-$\{item.colorClass || 'primary'}\` | **High** - Classes generated at runtime. | Use a lookup table. |
| `src/components/GenericCommandPalette.tsx` | 94 | \`text-$\{item.colorClass || 'primary'}/50\` | **High** - Classes generated at runtime. | Use a lookup table. |

### 🟠 Moderate Risk: Conditional Template Literals
These use full class strings within ternary operators inside template literals. While Tailwind usually detects these if the full string is present, complex nesting or dynamic logic can sometimes lead to misses in specific optimizer configurations.

| File | Line | Pattern | Risk | Recommendation |
| :--- | :--- | :--- | :--- | :--- |
| `src/lib/AlephTable.tsx` | 45 | \`${selectedIdx === i ? 'bg-primary/10 text-primary' : 'text-text hover:bg-surfaceAlt'}\` | Low-Medium | Generally safe, but verify in production build. |
| `src/components/DataPanel.tsx` | 25 | \`${visible ? 'translate-x-0' : 'translate-x-full'}\` | Low-Medium | Safe. |
| `src/components/OracleView.tsx` | 153 | \`${pred.predictedState === 'ACTION_REQUIRED' ? 'bg-warning/10 text-warning' : 'bg-primary/10 text-primary'}\` | Low-Medium | Safe. |
| `src/components/OracleView.tsx` | 186 | \`${feedbackGiven[pred.entityId] === true ? 'bg-success/10 text-success' : 'hover:bg-success/10 text-textDim hover:text-success'}\` | Low-Medium | Safe. |
| `src/components/OracleView.tsx` | 194 | \`${feedbackGiven[pred.entityId] === false ? 'bg-danger/10 text-danger' : 'hover:bg-danger/10 text-textDim hover:text-danger'}\` | Low-Medium | Safe. |
| `src/components/OracleView.tsx` | 236 | \`${sentimentResult.label === 'positive' ? 'bg-success/10 text-success' : ...}\` | Low-Medium | Safe. |
| `src/components/OracleView.tsx` | 244 | \`${sentimentResult.label === 'positive' ? 'text-success' : ...}\` | Low-Medium | Safe. |
| `src/components/SkillForm.tsx` | 104 | \`${isSaving ? 'opacity-50 cursor-not-allowed' : ''}\` | Low-Medium | Safe. |
| `src/components/SkillForm.tsx` | 111 | \`${isSaving ? 'opacity-50 cursor-not-allowed' : ''}\` | Low-Medium | Safe. |
| `src/components/forms/DataSourceFormSlideOver.tsx` | 100 | \`${step === s ? 'bg-primary' : 'bg-border'}\` | Low-Medium | Safe. |
| `src/components/CopilotView.tsx` | 122 | \`${splitView ? 'text-primary bg-primary/10' : 'text-textMuted hover:text-text'}\` | Low-Medium | Safe. |
| `src/components/CopilotView.tsx` | 166 | \`${splitView ? 'max-w-1/2' : 'w-full'}\` | Low-Medium | Safe. |
| `src/components/DataSourcesView.tsx` | 56 | \`${task.status === 'running' ... ? 'bg-warning/10 text-warning' : ...}\` | Low-Medium | Safe. |
| `src/components/DataSourcesView.tsx` | 77 | \`${(task.status === 'running' ...) ? 'bg-border text-textMuted' : ...}\` | Low-Medium | Safe. |
| `src/components/forms/SandboxResultSlideOver.tsx` | 10 | \`${result?.exitCode === 0 ? 'text-success' : 'text-danger'}\` | Low-Medium | Safe. |
| `src/components/terminal/TerminalPrompt.tsx` | 100 | \`${idx === completionIndex ? 'bg-primary text-background' : 'bg-surface-alt text-text'}\` | Low-Medium | Safe. |
| `src/components/ExplorerView.tsx` | 36 | \`${selectedObject === obj ? 'bg-primary/10 text-primary border-primary/30' : ...}\` | Low-Medium | Safe. |
| `src/components/ExplorerView.tsx` | 55-58 | \`${activeView === 'table' ? 'bg-primary/10 text-primary' : ...}\` | Low-Medium | Safe. |

### ⚪ Low Risk: Non-Tailwind Dynamic Classes
These use dynamic strings for attributes other than CSS classes or for non-Tailwind identifier strings.

| File | Line | Pattern | Risk | Note |
| :--- | :--- | :--- | :--- | :--- |
| `src/utils/fuzzySearch.tsx` | 79 | \`className={highlightClass}\` | None | `highlightClass` is passed as a prop (full string). |
| `src/components/terminal/TerminalProgressBar.tsx` | 91 | \`${colorClass} opacity-60\` | None | `colorClass` is typically a full Tailwind class string. |

---

## Safelist Analysis
The `tailwind.config.js` file does **not** currently contain a `safelist` property. 

## Final Recommendations
1. **Urgent:** Refactor `GenericCommandPalette.tsx` to use a mapping object for `colorClass` (e.g., `const colorMap = { primary: 'text-primary', warning: 'text-warning' }`) instead of string interpolation.
2. **Verification:** Since most other patterns use full strings in ternaries, they are likely safe, but a production build should be manually verified for the "Moderate Risk" items.
