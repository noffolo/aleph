import { describe, it, expect } from 'vitest'
import { t } from '../index'

describe('i18n t()', () => {
  it('returns translated string for known key', () => {
    expect(t('agents.create')).toBe('Nuovo Agente')
  })

  it('returns key itself for unknown key', () => {
    expect(t('nonexistent.key')).toBe('nonexistent.key')
  })

  it('returns key for empty string', () => {
    expect(t('')).toBe('')
  })

  it('interpolates known key with param', () => {
    // 'app.searchPrompt' = "Digita per cercare {title}..."
    expect(t('app.searchPrompt', { title: 'agenti' })).toBe('Digita per cercare agenti...')
  })

  it('interpolates known key with multiple params', () => {
    expect(t('commandPalette.genericPrompt', { title: 'comandi' })).toBe('Digita per cercare comandi...')
  })

  it('interpolates single param (unknown key falls through)', () => {
    expect(t('hello {name}', { name: 'World' })).toBe('hello {name}')
  })

  it('returns key itself for unknown key with params (no match)', () => {
    expect(t('foo {bar}', { bar: 'baz' })).toBe('foo {bar}')
  })

  it('returns key when params is empty object', () => {
    expect(t('unknown', {})).toBe('unknown')
  })
})
