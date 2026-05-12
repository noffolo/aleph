import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ComponentFormSlideOver } from '../ComponentFormSlideOver'

vi.mock('../../../hooks/domain/useComponentActions', () => ({
  useComponentActions: () => ({
    onRegisterComponent: vi.fn(),
  }),
}))

vi.mock('../../../schemas', () => ({
  RegistryComponentSchema: { safeParse: vi.fn(() => ({ success: true, data: {} })) },
}))

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'components.register': 'Registra Componente',
      'components.form.name': 'Nome componente',
      'components.form.description': 'Descrizione',
      'components.form.systemPrompt': 'Prompt di sistema',
      'confirmDialog.cancel': 'Annulla',
    }
    return map[key] ?? key
  },
}))

describe('ComponentFormSlideOver', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders title', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    expect(screen.getByRole('heading', { name: 'Registra Componente' })).toBeInTheDocument()
  })

  it('renders name and type fields', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    expect(screen.getByLabelText('Nome')).toBeInTheDocument()
    expect(screen.getByLabelText('Tipo')).toBeInTheDocument()
  })

  it('renders category and source selects', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    expect(screen.getByLabelText('Categoria')).toBeInTheDocument()
    expect(screen.getByLabelText('Sorgente')).toBeInTheDocument()
  })

  it('renders status and approval selects', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    expect(screen.getByLabelText('Stato')).toBeInTheDocument()
    expect(screen.getByLabelText('Approvazione')).toBeInTheDocument()
  })

  it('renders JSON config fields', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    expect(screen.getByLabelText('Schema Config (JSON)')).toBeInTheDocument()
    expect(screen.getByLabelText('Dependencies (JSON)')).toBeInTheDocument()
  })

  it('renders cancel and submit buttons', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    expect(screen.getByText('Annulla')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Registra Componente' })).toBeInTheDocument()
  })

  it('calls onClose on cancel', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    fireEvent.click(screen.getByText('Annulla'))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it('shows error for empty name on submit', () => {
    render(<ComponentFormSlideOver onClose={mockOnClose} />)
    fireEvent.click(screen.getByRole('button', { name: 'Registra Componente' }))
    expect(screen.getByText('Il nome è obbligatorio')).toBeInTheDocument()
  })
})
