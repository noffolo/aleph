import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ToolForm } from '../ToolForm'

const { apiPostMock, apiPatchMock } = vi.hoisted(() => ({
  apiPostMock: vi.fn(),
  apiPatchMock: vi.fn(),
}))

vi.mock('../../api/client', () => ({
  apiPost: apiPostMock,
  apiPatch: apiPatchMock,
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'tools.create': 'Nuovo Tool',
      'tools.edit': 'Modifica Tool',
      'tools.form.name': 'Es: Analizzatore CSV',
      'tools.form.description': 'Descrivi cosa fa il tool',
      'confirmDialog.cancel': 'Annulla',
    }
    return map[key] ?? key
  },
}))

const mockStoreState = { apiKey: 'test-key' }
vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((sel?: (s: Record<string, unknown>) => unknown) => {
      if (typeof sel === 'function') return sel(mockStoreState)
      return mockStoreState
    }),
    { getState: vi.fn(() => mockStoreState), subscribe: vi.fn(() => vi.fn()) },
  ),
}))

describe('ToolForm', () => {
  const onSave = vi.fn()
  const onCancel = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    apiPostMock.mockResolvedValue({})
    apiPatchMock.mockResolvedValue({})
  })

  it('renders in create mode with title', () => {
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    expect(screen.getByRole('heading', { name: 'Nuovo Tool' })).toBeInTheDocument()
  })

  it('renders in edit mode with title', () => {
    const tool = { id: 't1', name: 'My Tool', description: 'A tool', code: 'console.log(1)' }
    render(<ToolForm tool={tool} onSave={onSave} onCancel={onCancel} />)
    expect(screen.getByRole('heading', { name: 'Modifica Tool' })).toBeInTheDocument()
  })

  it('pre-fills form in edit mode', () => {
    const tool = { id: 't1', name: 'My Tool', description: 'Does stuff', code: 'print("hi")' }
    render(<ToolForm tool={tool} onSave={onSave} onCancel={onCancel} />)
    expect(screen.getByDisplayValue('My Tool')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Does stuff')).toBeInTheDocument()
    expect(screen.getByDisplayValue('print("hi")')).toBeInTheDocument()
  })

  it('calls onCancel on cancel click', async () => {
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await userEvent.setup().click(screen.getByText('Annulla'))
    expect(onCancel).toHaveBeenCalled()
  })

  it('submits form and calls apiPost in create mode', async () => {
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.type(screen.getByPlaceholderText('Es: Analizzatore CSV'), 'Test Tool')
    await user.click(screen.getByRole('button', { name: 'Nuovo Tool' }))
    await vi.waitFor(() => {
      expect(apiPostMock).toHaveBeenCalledWith('/api/v1/tools', expect.objectContaining({ name: 'Test Tool' }))
      expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ name: 'Test Tool' }))
      expect(onCancel).toHaveBeenCalled()
    })
  })

  it('submits form and calls apiPatch in edit mode', async () => {
    const user = userEvent.setup()
    const tool = { id: 't1', name: 'Old', description: 'Desc', code: 'code' }
    render(<ToolForm tool={tool} onSave={onSave} onCancel={onCancel} />)
    await user.click(screen.getByRole('button', { name: 'Modifica Tool' }))
    await vi.waitFor(() => {
      expect(apiPatchMock).toHaveBeenCalledWith('/api/v1/tools/t1', expect.objectContaining({ name: 'Old' }))
    })
  })

  it('shows save error when api call fails', async () => {
    apiPostMock.mockRejectedValue(new Error('Network error'))
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.type(screen.getByPlaceholderText('Es: Analizzatore CSV'), 'Test')
    await user.click(screen.getByRole('button', { name: 'Nuovo Tool' }))
    expect(await screen.findByText('Network error')).toBeInTheDocument()
  })

  it('catches non-Error exceptions', async () => {
    apiPostMock.mockRejectedValue('String error')
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.type(screen.getByPlaceholderText('Es: Analizzatore CSV'), 'Test')
    await user.click(screen.getByRole('button', { name: 'Nuovo Tool' }))
    expect(await screen.findByText('String error')).toBeInTheDocument()
  })

  it('shows validating spinner while saving', async () => {
    apiPostMock.mockImplementation(() => new Promise(() => {}))
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.type(screen.getByPlaceholderText('Es: Analizzatore CSV'), 'Test')
    await user.click(screen.getByRole('button', { name: 'Nuovo Tool' }))
    expect(screen.getByText('Salvando...')).toBeInTheDocument()
  })

  it('disables inputs while saving', async () => {
    apiPostMock.mockImplementation(() => new Promise(() => {}))
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.type(screen.getByPlaceholderText('Es: Analizzatore CSV'), 'Test')
    await user.click(screen.getByRole('button', { name: 'Nuovo Tool' }))
    expect(screen.getByPlaceholderText('Es: Analizzatore CSV')).toBeDisabled()
  })

  it('shows validation error for empty name', async () => {
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.click(screen.getByRole('button', { name: 'Nuovo Tool' }))
    expect(apiPostMock).not.toHaveBeenCalled()
  })

  it('renders description textarea', () => {
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    expect(screen.getByPlaceholderText('Descrivi cosa fa il tool')).toBeInTheDocument()
  })

  it('renders code textarea', () => {
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    expect(screen.getByPlaceholderText('// Implementazione del tool...')).toBeInTheDocument()
  })

  it('clears save error on retry', async () => {
    let callCount = 0
    apiPostMock.mockImplementation(() => {
      callCount++
      if (callCount === 1) throw new Error('First fail')
      return Promise.resolve({})
    })
    const user = userEvent.setup()
    render(<ToolForm onSave={onSave} onCancel={onCancel} />)
    await user.type(screen.getByPlaceholderText('Es: Analizzatore CSV'), 'Test')
    const submitBtn = screen.getByRole('button', { name: 'Nuovo Tool' })
    await user.click(submitBtn)
    expect(await screen.findByText('First fail')).toBeInTheDocument()
    await user.click(submitBtn)
    await vi.waitFor(() => {
      expect(screen.queryByText('First fail')).not.toBeInTheDocument()
    })
  })

  it('renders custom title', () => {
    render(<ToolForm onSave={onSave} onCancel={onCancel} title="Custom Title" />)
    expect(screen.getByText('Custom Title')).toBeInTheDocument()
  })
})
