/**
 * Scene ↔ View mapping tables for W2 Navigation Redesign.
 *
 * Used by NavigationStateSync (URL → store sync), useAppActions
 * (SHOW_INLINE rerouting), and scene components (view dispatch).
 */

export const VIEW_TO_SCENE: Record<string, string | null> = {
  explore:   'explore',
  library:   'explore',
  ontology:  'explore',
  data:      'explore',
  agent:     'agents',
  skill:     'agents',
  tool:      'agents',
  component: 'agents',
  health:    'system',
  settings:  'system',
  predict:   'system',
  dashboard: 'terminal',
}

/** Views dispatched within the Explore scene. */
export const EXPLORE_VIEWS = ['explore', 'library', 'ontology', 'data']

/** Views dispatched within the Agents scene. */
export const AGENT_VIEWS = ['agent', 'skill', 'tool', 'component']

/** Views dispatched within the System scene. */
export const SYSTEM_VIEWS = ['health', 'settings', 'predict']

/** Human-readable labels for view types. */
export const VIEW_LABELS: Record<string, string> = {
  explore:    'Explorer',
  library:    'Library',
  ontology:   'Ontologies',
  data:       'Data Sources',
  agent:      'Agents',
  skill:      'Skills',
  tool:       'Tools',
  component:  'Components',
  health:     'Data Health',
  settings:   'Settings',
  predict:    'Oracle',
  dashboard:  'Dashboard',
}
