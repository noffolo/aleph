import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ComponentDetailSlideOver } from '../ComponentDetailSlideOver'

const mockComponents = [
  {
    id: 'c1',
    name: 'Analyzer',
    description: 'Does analysis',
    version: '1.0.0',
    type: 'tool',
    category: 'analytical',
    source: 'registry',
    status: 'active',
    approvalStatus: 'approved',
    promptTemplate: 'You are an analyzer',
  },
]

vi.mock('../../../store/useStore', () => ({
  useStore: vi.fn((selector?: (state: unknown) => unknown) => {
    const state = { registryComponents: mockComponents }
    if (typeof selector === 'function') return selector(state)
    return state
  }),
}))

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'slideOver.close': 'Chiudi',
    }
    return map[key] ?? key
  },
}))

describe('ComponentDetailSlideOver', () => {
  const mockOnClose = vi.fn()

  it('renders component name', () => {
    render(<ComponentDetailSlideOver componentId="c1" onClose={mockOnClose} />)
    expect(screen.getByText('Analyzer')).toBeInTheDocument()
  })

  it('renders component description', () => {
    render(<ComponentDetailSlideOver componentId="c1" onClose={mockOnClose} />)
    expect(screen.getByText('Does analysis')).toBeInTheDocument()
  })

  it('renders metadata fields (type, category, source, status)', () => {
    render(<ComponentDetailSlideOver componentId="c1" onClose={mockOnClose} />)
    expect(screen.getByText('tool')).toBeInTheDocument()
    expect(screen.getByText('analytical')).toBeInTheDocument()
    expect(screen.getByText('registry')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
  })

  it('renders version', () => {
    render(<ComponentDetailSlideOver componentId="c1" onClose={mockOnClose} />)
    expect(screen.getByText('1.0.0')).toBeInTheDocument()
  })

  it('renders prompt template when present', () => {
    render(<ComponentDetailSlideOver componentId="c1" onClose={mockOnClose} />)
    expect(screen.getByText('You are an analyzer')).toBeInTheDocument()
  })

  it('returns null for non-existent component', () => {
    const { container } = render(<ComponentDetailSlideOver componentId="nonexistent" onClose={mockOnClose} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders close button', () => {
    render(<ComponentDetailSlideOver componentId="c1" onClose={mockOnClose} />)
    expect(screen.getByText('Chiudi')).toBeInTheDocument()
  })
})
