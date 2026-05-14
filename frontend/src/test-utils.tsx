import { vi } from 'vitest'

export function mockI18n(overrides?: Record<string, string>) {
  const defaults: Record<string, string> = {
    'agents.create': 'Nuovo Agente',
    'agents.edit': 'Modifica Agente',
    'agents.form.name': 'Es: Analista Finanze',
    'agents.form.model': 'Es: gpt-4o-mini o llama3.2',
    'agents.form.apiKey': 'Inserisci solo per sovrascrivere la chiave esistente (facoltativo)',
    'agents.form.baseUrl': 'Es: https://api.openai.com/v1',
    'agents.form.systemPrompt': "Definisci il ruolo dell'agente",
    'skills.create': 'Crea Skill',
    'skills.edit': 'Modifica Skill',
    'skills.form.name': 'Es: Analista Finanze',
    'skills.form.description': 'Descrivi la capacita di questa skill...',
    'skills.form.nameRequired': 'Il nome e obbligatorio',
    'confirmDialog.cancel': 'Annulla',
    'common.search': 'Cerca...',
    'common.save': 'Salva',
    'common.delete': 'Elimina',
    'common.loading': 'Caricamento...',
    'oracle.title': 'Oracolo',
    'datasource.form.title': 'Nuova Sorgente',
    'datasource.form.editTitle': 'Modifica Sorgente',
    'datasource.form.name': 'Nome Sorgente',
    'datasource.form.type': 'Tipo Sorgente',
    'datasource.form.format': 'Formato File',
    'datasource.form.url': 'URL API',
    'datasource.form.connectionString': 'Stringa di Connessione',
    ...overrides,
  }
  vi.mock('../../i18n', () => ({
    t: (key: string) => defaults[key] ?? key,
  }))
}

export function mockStore(state: Record<string, unknown> = {}) {
  const getState = vi.fn(() => state)
  vi.mock('../store/useStore', () => ({
    useStore: Object.assign(
      vi.fn((sel: (s: typeof state) => unknown) => sel(getState())),
      { subscribe: vi.fn(() => vi.fn()), getState }
    ),
  }))
}

export function mockFeatures(enabled: string[] = []) {
  vi.mock('../config/features', () => {
    const flags: Record<string, boolean> = {}
    for (const f of enabled) flags[f] = true
    return {
      isEnabled: vi.fn((key: string) => !!flags[key]),
      ...Object.fromEntries(Object.keys(flags).map(k => [k, flags[k]])),
    }
  })
}

export function mockNuqs(value = '') {
  vi.mock('nuqs', () => ({
    useQueryState: vi.fn(() => [value, vi.fn()]),
  }))
}

export function jsdomPolyfills() {
  Element.prototype.scrollTo = vi.fn() as unknown as typeof Element.prototype.scrollTo
  class MockIntersectionObserver {
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
  }
  Object.defineProperty(window, 'IntersectionObserver', {
    writable: true, configurable: true, value: MockIntersectionObserver,
  })
  class MockResizeObserver {
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
  }
  Object.defineProperty(window, 'ResizeObserver', {
    writable: true, configurable: true, value: MockResizeObserver,
  })
  window.HTMLElement.prototype.scrollIntoView = vi.fn()
}
