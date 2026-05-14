import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SkillForm } from '../SkillForm'

const { safeParseMock } = vi.hoisted(() => ({
  safeParseMock: vi.fn(() => ({ success: true, data: {} })),
}))

vi.mock('../../schemas', () => ({
  SkillFormSchema: {
    safeParse: safeParseMock,
  },
}))

vi.mock('../../api/client', () => ({
  apiPost: vi.fn(),
  apiPatch: vi.fn(),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'skills.create': 'Crea Skill',
      'skills.edit': 'Modifica Skill',
      'skills.form.name': 'Es: Analista Finanze',
      'skills.form.description': 'Descrivi la capacita di questa skill...',
      'confirmDialog.cancel': 'Annulla',
    }
    return map[key] ?? key
  },
}))

const mockTools = [
  { id: 'tool-1', name: 'Sentiment Analyzer', description: 'desc', code: '...' },
  { id: 'tool-2', name: 'Data Fetcher', description: 'desc', code: '...' },
  { id: 'tool-3', name: 'Image Generator', description: 'desc', code: '...' },
]

const mockSkill = {
  id: 'skill-1',
  name: 'Financial Analyst',
  description: 'Analyzes financial data and generates reports',
  toolIds: ['tool-1'],
}

describe('SkillForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    safeParseMock.mockReturnValue({ success: true, data: {} })
  })

  it('renders create mode with title', () => {
    render(<SkillForm tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByRole('heading', { name: 'Crea Skill' })).toBeInTheDocument()
  })

  it('renders edit mode with title', () => {
    render(<SkillForm skill={mockSkill} tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByText('Modifica Skill')).toBeInTheDocument()
  })

  it('renders custom title', () => {
    render(<SkillForm tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} title="Custom Title" />)
    expect(screen.getByText('Custom Title')).toBeInTheDocument()
  })

  it('pre-fills name in edit mode', () => {
    render(<SkillForm skill={mockSkill} tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByDisplayValue('Financial Analyst')).toBeInTheDocument()
  })

  it('renders tool checkboxes', () => {
    render(<SkillForm tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    expect(screen.getByLabelText('Sentiment Analyzer')).toBeInTheDocument()
  })

  it('pre-checks associated tools', () => {
    render(<SkillForm skill={mockSkill} tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    expect((screen.getByLabelText('Sentiment Analyzer') as HTMLInputElement).checked).toBe(true)
  })

  it('calls onCancel on cancel click', async () => {
    const onCancel = vi.fn()
    render(<SkillForm tools={mockTools} onSave={vi.fn()} onCancel={onCancel} />)
    await userEvent.setup().click(screen.getByText('Annulla'))
    expect(onCancel).toHaveBeenCalled()
  })

  it('shows validation error when schema fails', async () => {
    safeParseMock.mockReturnValue({ success: false, error: { issues: [{ path: ['name'], message: 'Required' }] } })
    render(<SkillForm tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Crea Skill' }))
    expect(screen.getByText('Required')).toBeInTheDocument()
  })

  it('submits via apiPost in create mode', async () => {
    const { apiPost } = await import('../../api/client')
    const onSave = vi.fn()
    const onCancel = vi.fn()
    render(<SkillForm tools={mockTools} onSave={onSave} onCancel={onCancel} />)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Crea Skill' }))
    expect(apiPost).toHaveBeenCalledWith('/api/v1/skills', expect.any(Object))
  })

  it('submits via apiPatch in edit mode', async () => {
    const { apiPatch } = await import('../../api/client')
    const onSave = vi.fn()
    const onCancel = vi.fn()
    render(<SkillForm skill={mockSkill} tools={mockTools} onSave={onSave} onCancel={onCancel} />)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Aggiorna Skill' }))
    expect(apiPatch).toHaveBeenCalledWith('/api/v1/skills/skill-1', expect.any(Object))
  })

  it('shows save error when api call fails', async () => {
    const { apiPost } = await import('../../api/client');
    (apiPost as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))
    render(<SkillForm tools={mockTools} onSave={vi.fn()} onCancel={vi.fn()} />)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Crea Skill' }))
    expect(await screen.findByText('Network error')).toBeInTheDocument()
  })
})
