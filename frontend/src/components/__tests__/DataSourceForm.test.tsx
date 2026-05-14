import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DataSourceForm } from '../DataSourceForm'

const { safeParseMock } = vi.hoisted(() => ({
  safeParseMock: vi.fn(() => ({ success: true, data: {} })),
}))

vi.mock('../../schemas', () => ({
  DataSourceFormSchema: {
    safeParse: safeParseMock,
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
    safeParseMock.mockReturnValue({ success: true, data: {} })
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
    expect(screen.getByText('DB')).toBeInTheDocument()
  })

  it('switches to API mode', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('API'))
    expect(screen.getByText('URL Endpoint')).toBeInTheDocument()
  })

  it('switches to DB mode', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('DB'))
    expect(screen.getByText('Connetti Database')).toBeInTheDocument()
  })

  it('shows file format selector in file mode', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByText('Formato File')).toBeInTheDocument()
    expect(screen.getByText('CSV')).toBeInTheDocument()
    expect(screen.getByText('JSON')).toBeInTheDocument()
    expect(screen.getByText('XML')).toBeInTheDocument()
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

  it('calls onSave when validation passes', () => {
    safeParseMock.mockReturnValue({ success: true, data: {} })
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('Crea Sorgente'))
    expect(mockOnSave).toHaveBeenCalledTimes(1)
  })

  it('does not call onSave when validation fails', () => {
    safeParseMock.mockReturnValue({
      success: false,
      error: { issues: [{ path: ['name'], message: 'Required' }] },
    })
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('Crea Sorgente'))
    expect(mockOnSave).not.toHaveBeenCalled()
  })

  it('shows validation error on name field', () => {
    safeParseMock.mockReturnValue({
      success: false,
      error: { issues: [{ path: ['name'], message: 'Name is required' }] },
    })
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('Crea Sorgente'))
    expect(screen.getByText('Name is required')).toBeInTheDocument()
  })

  it('shows configJson error for invalid API URL when URL is non-empty but invalid', () => {
    safeParseMock.mockReturnValue({
      success: false,
      error: { issues: [] },
    })
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('API'))
    const urlInput = screen.getByPlaceholderText('https://api.example.com/v1/data')
    fireEvent.change(urlInput, { target: { value: 'not-a-valid-url' } })
    fireEvent.click(screen.getByText('Crea Sorgente'))
    expect(screen.getByText('URL in config must start with http:// or https://')).toBeInTheDocument()
  })

  it('shows configJson error for empty DB connection string', () => {
    safeParseMock.mockReturnValue({
      success: false,
      error: { issues: [] },
    })
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('DB'))
    fireEvent.click(screen.getByText('Crea Sorgente'))
    expect(screen.getByText('connectionString is required in config JSON')).toBeInTheDocument()
  })

  it('hides file format selector after switching to API', () => {
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    expect(screen.getByText('Formato File')).toBeInTheDocument()
    fireEvent.click(screen.getByText('API'))
    expect(screen.queryByText('Formato File')).not.toBeInTheDocument()
  })

  it('shows valid API URL passes validation', () => {
    safeParseMock.mockReturnValue({ success: true, data: {} })
    render(<DataSourceForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    fireEvent.click(screen.getByText('API'))
    fireEvent.click(screen.getByText('Crea Sorgente'))
    expect(mockOnSave).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('URL in config must start with http:// or https://')).not.toBeInTheDocument()
  })
})
