import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ToolExecuteSlideOver } from '../ToolExecuteSlideOver'

vi.mock('../../../store/useStore', () => ({
  useStore: vi.fn((selector?: (state: unknown) => unknown) => {
    const state: Record<string, unknown> = {
      sandboxInput: '{}',
      setSandboxInput: mockSetSandboxInput,
    }
    if (typeof selector === 'function') return selector(state)
    return state
  }),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useToolActions', () => ({
  useToolActions: () => ({ onExecuteTool: vi.fn() }),
}))

const mockSetSandboxInput = vi.fn()

describe('ToolExecuteSlideOver', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const tool = { id: 't1', name: 'SqlRunner', description: 'Runs SQL queries', code: 'SELECT * FROM users' }

  it('renders tool name as title', () => {
    render(<ToolExecuteSlideOver tool={tool} />)
    expect(screen.getByText('SqlRunner')).toBeInTheDocument()
  })

  it('renders tool description', () => {
    render(<ToolExecuteSlideOver tool={tool} />)
    expect(screen.getByText('Runs SQL queries')).toBeInTheDocument()
  })

  it('renders code preview', () => {
    render(<ToolExecuteSlideOver tool={tool} />)
    expect(screen.getByText('SELECT * FROM users')).toBeInTheDocument()
  })

  it('renders JSON input textarea', () => {
    render(<ToolExecuteSlideOver tool={tool} />)
    expect(screen.getByDisplayValue('{}')).toBeInTheDocument()
  })

  it('calls setSandboxInput on textarea change', () => {
    render(<ToolExecuteSlideOver tool={tool} />)
    const textarea = screen.getByDisplayValue('{}')
    fireEvent.change(textarea, { target: { value: '{"query": "test"}' } })
    expect(mockSetSandboxInput).toHaveBeenCalledWith('{"query": "test"}')
  })

  it('renders execute button', () => {
    render(<ToolExecuteSlideOver tool={tool} />)
    expect(screen.getByText('Esegui Tool nel Sandbox')).toBeInTheDocument()
  })

  it('returns null for tool without id', () => {
    const { container } = render(<ToolExecuteSlideOver tool={{ id: '', name: '', description: '', code: '' }} />)
    expect(container.firstChild).toBeNull()
  })
})
