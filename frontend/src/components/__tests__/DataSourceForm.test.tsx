import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DataSourceForm } from '../DataSourceForm'

vi.mock('../../schemas', () => ({
  DataSourceFormSchema: {
    safeParse: vi.fn(() => ({ success: true, data: {} })),
  },
}))

vi.mock('lucide-react', () => ({
  Upload: () => null,
  Globe: () => null,
  Database: () => null,
  FileText: () => null,
  Code: () => null,
  Link: () => null,
}))

describe('DataSourceForm', () => {
  const mockOnSave = vi.fn()
  const mockOnCancel = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders title', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByText('Nuova Sorgente Dati')).toBeInTheDocument()
  })

  it('renders custom title', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} title="Custom Title" />)
    expect(screen.getByText('Custom Title')).toBeInTheDocument()
  })

  it('renders name input', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByLabelText('Nome')).toBeInTheDocument()
  })

  it('renders mode selector buttons (File, API, DB)', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByText('File')).toBeInTheDocument()
    expect(screen.getByText('API')).toBeInTheDocument()
  })

  it('switches to API mode', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('API'))
    expect(screen.getByText('URL Endpoint')).toBeInTheDocument()
  })

  it('renders config JSON textarea', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByLabelText('Configurazione Avanzata (JSON)')).toBeInTheDocument()
  })

  it('renders cancel and submit buttons', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByText('Annulla')).toBeInTheDocument()
    expect(screen.getByText('Crea Sorgente')).toBeInTheDocument()
  })

  it('calls onCancel on cancel click', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('Annulla'))
    expect(mockOnCancel).toHaveBeenCalledTimes(1)
  })
})
