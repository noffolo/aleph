import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ToolFormSlideOver } from '../ToolFormSlideOver'

vi.mock('../../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn(() => ({})),
    { getState: vi.fn(() => ({ setSlideOverContent: vi.fn() })) },
  ),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useToolActions', () => ({
  useToolActions: () => ({
    onCreateTool: vi.fn().mockResolvedValue(undefined),
    onUpdateTool: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('../../../schemas', () => ({
  ToolSchema: { safeParse: vi.fn(() => ({ success: true, data: {} })) },
}))

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'tools.create': 'Crea Tool',
      'tools.edit': 'Modifica Tool',
      'tools.form.name': 'Nome tool',
      'tools.form.description': 'Descrizione',
      'confirmDialog.cancel': 'Annulla',
      'generic.saving': 'Salvataggio...',
    }
    return map[key] ?? key
  },
}))

describe('ToolFormSlideOver', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders create title when no tool provided', () => {
    render(<ToolFormSlideOver />)
    expect(screen.getByRole('heading', { name: 'Crea Tool' })).toBeInTheDocument()
  })

  it('renders edit title when tool provided', () => {
    const tool = { id: 't1', name: 'Test Tool', description: 'desc', code: 'return 1' }
    render(<ToolFormSlideOver tool={tool} />)
    expect(screen.getByRole('heading', { name: 'Modifica Tool' })).toBeInTheDocument()
  })

  it('renders name, description, and code fields', () => {
    render(<ToolFormSlideOver />)
    expect(screen.getByLabelText('Nome')).toBeInTheDocument()
    expect(screen.getByLabelText('Descrizione')).toBeInTheDocument()
    expect(screen.getByLabelText('Codice')).toBeInTheDocument()
  })

  it('pre-fills fields in edit mode', () => {
    const tool = { id: 't1', name: 'MyTool', description: 'Does stuff', code: 'console.log("hi")' }
    render(<ToolFormSlideOver tool={tool} />)
    expect(screen.getByDisplayValue('MyTool')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Does stuff')).toBeInTheDocument()
    expect(screen.getByDisplayValue('console.log("hi")')).toBeInTheDocument()
  })

  it('renders cancel and submit buttons', () => {
    render(<ToolFormSlideOver />)
    expect(screen.getByText('Annulla')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Crea Tool' })).toBeInTheDocument()
  })
})
