import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DataSourceFormSlideOver } from '../DataSourceFormSlideOver'

vi.mock('../../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn(() => ({})),
    { getState: vi.fn(() => ({ setSlideOverContent: vi.fn() })) },
  ),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useDataSourceActions', () => ({
  useDataSourceActions: () => ({
    onAddSource: vi.fn(),
  }),
}))

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'datasources.title': 'Sorgenti Dati',
      'datasources.form.name': 'Nome sorgente',
      'datasources.form.description': 'Descrizione',
      'confirmDialog.cancel': 'Annulla',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => ({
  Upload: () => null,
  Globe: () => null,
  Database: () => null,
  FileText: () => null,
  Code: () => null,
  Link: () => null,
  ChevronLeft: () => null,
  ChevronRight: () => null,
  Check: () => null,
}))

describe('DataSourceFormSlideOver', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders title', () => {
    render(<DataSourceFormSlideOver />)
    expect(screen.getByText('Nuova Sorgente Dati')).toBeInTheDocument()
  })

  it('starts at step 1 with name field', () => {
    render(<DataSourceFormSlideOver />)
    expect(screen.getByLabelText('Nome Sorgente')).toBeInTheDocument()
  })

  it('advances to step 2 on next click', () => {
    render(<DataSourceFormSlideOver />)
    fireEvent.change(screen.getByLabelText('Nome Sorgente'), { target: { value: 'Test Source' } })
    fireEvent.click(screen.getByText('Avanti'))
    expect(screen.getByText('Scegli Tipo Sorgente')).toBeInTheDocument()
  })

  it('advances to step 3 with next after mode selection', () => {
    render(<DataSourceFormSlideOver />)
    fireEvent.change(screen.getByLabelText('Nome Sorgente'), { target: { value: 'Test Source' } })
    fireEvent.click(screen.getByText('Avanti'))
    fireEvent.click(screen.getByText('Avanti'))
    expect(screen.getByLabelText('Configurazione Avanzata (JSON)')).toBeInTheDocument()
  })

  it('selects API mode and shows URL input on step 3', () => {
    render(<DataSourceFormSlideOver />)
    fireEvent.change(screen.getByLabelText('Nome Sorgente'), { target: { value: 'Test Source' } })
    fireEvent.click(screen.getByText('Avanti'))
    fireEvent.click(screen.getByText('API'))
    fireEvent.click(screen.getByText('Avanti'))
    expect(screen.getByLabelText('URL Endpoint')).toBeInTheDocument()
  })

  it('shows error when submitting without name', () => {
    render(<DataSourceFormSlideOver />)
    fireEvent.click(screen.getByText('Avanti'))
    expect(screen.getByText('Il nome è obbligatorio')).toBeInTheDocument()
  })

  it('renders step indicators', () => {
    render(<DataSourceFormSlideOver />)
    const indicators = document.querySelectorAll('.rounded-full')
    expect(indicators.length).toBeGreaterThanOrEqual(3)
  })
})
