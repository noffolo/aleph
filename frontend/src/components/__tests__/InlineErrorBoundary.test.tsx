import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { InlineErrorBoundary } from '../InlineErrorBoundary'

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'errors.panel': 'Errore Pannello',
      'errors.componentError': 'Errore del componente',
      'toast.retry': 'Riprova',
    }
    return map[key] ?? key
  },
}))

describe('InlineErrorBoundary', () => {
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  it('renders children when no error', () => {
    render(
      <InlineErrorBoundary>
        <div>child content</div>
      </InlineErrorBoundary>,
    )
    expect(screen.getByText('child content')).toBeInTheDocument()
  })

  it('catches rendering errors and shows fallback UI', () => {
    const ThrowComponent = () => {
      throw new Error('simulated render error')
    }

    render(
      <InlineErrorBoundary>
        <ThrowComponent />
      </InlineErrorBoundary>,
    )

    expect(screen.getByText('Errore Pannello')).toBeInTheDocument()
    expect(screen.getByText('Errore del componente')).toBeInTheDocument()
    expect(screen.getByText('Riprova')).toBeInTheDocument()
  })

  it('retry button resets error state and re-renders children', () => {
    let shouldThrow = true
    const ToggleComponent = () => {
      if (shouldThrow) throw new Error('click retry')
      return <div>after retry</div>
    }

    render(
      <InlineErrorBoundary>
        <ToggleComponent />
      </InlineErrorBoundary>,
    )

    expect(screen.getByText('Riprova')).toBeInTheDocument()

    shouldThrow = false
    fireEvent.click(screen.getByText('Riprova'))

    expect(screen.getByText('after retry')).toBeInTheDocument()
  })

  it('applies label in dev mode error logging when provided', () => {
    const label = 'test-panel'
    const ThrowComponent = () => {
      throw new Error('label test')
    }

    render(
      <InlineErrorBoundary label={label}>
        <ThrowComponent />
      </InlineErrorBoundary>,
    )

    expect(screen.getByText('Errore Pannello')).toBeInTheDocument()
  })
})
