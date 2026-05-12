/**
 * Aleph UX Redeisgn — Feature flag (currently one flag: compact-sidebar).
 *
 * All flags default **disabled**. Enable at deploy time via
 * `VITE_FEATURE_*` environment variables without touching source.
 *
 * @module
 */

// ─── Flag Identifiers ──────────────────────────────────────

export type FeatureFlag =
  | 'compact-sidebar';

// ─── Metadata ──────────────────────────────────────────────

export interface FeatureFlagMeta {
  flag: FeatureFlag;
  label: string;
  description: string;
}

// ─── Default Values ───────────────────────────────────────

export const DefaultFeatures: Record<FeatureFlag, boolean> = {
  'compact-sidebar': false,
};

export const FeatureFlagMetas: Record<FeatureFlag, FeatureFlagMeta> = {
  'compact-sidebar': {
    flag: 'compact-sidebar',
    label: 'Compact Sidebar',
    description: 'Condense sidebar navigation to icons + tooltips.',
  },
};

// ─── Convenience Constants ─────────────────────────────────

export const FEATURE_COMPACT_SIDEBAR: FeatureFlag = 'compact-sidebar';

// ─── Runtime Check ─────────────────────────────────────────

export function flagEnvKey(flag: FeatureFlag): `VITE_FEATURE_${string}` {
  return `VITE_FEATURE_${flag.toUpperCase().replace(/-/g, '_')}` as const;
}

export type FlagContext = Record<string, unknown>;

export function isEnabled(flag: FeatureFlag, _ctx?: FlagContext): boolean {
  const envVar = flagEnvKey(flag);
  const envValue = import.meta.env[envVar];
  if (envValue !== undefined) {
    return envValue === 'true' || envValue === '1';
  }
  return DefaultFeatures[flag];
}

export function getAllFlags(): Array<{ flag: FeatureFlag; enabled: boolean }> {
  return (Object.keys(DefaultFeatures) as FeatureFlag[])
    .sort()
    .map((flag) => ({ flag, enabled: isEnabled(flag) }));
}
