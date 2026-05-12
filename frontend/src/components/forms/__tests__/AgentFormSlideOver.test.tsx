import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AgentFormSlideOver } from '../AgentFormSlideOver'

vi.mock('../../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn(() => ({})),
    { getState: vi.fn(() => ({ setSlideOverContent: vi.fn() })) },
  ),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useAgentActions', () => ({
  useAgentActions: () => ({
    onCreateAgent: vi.fn().mockResolvedValue(undefined),
    onUpdateAgent: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('../../../schemas', () => ({
  AgentSchema: { safeParse: vi.fn(() => ({ success: true, data: {} })) },
}))

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'agents.create': 'Crea Agente',
      'agents.edit': 'Modifica Agente',
      'agents.form.name': 'Nome agente',
      'agents.form.model': 'Modello',
      'agents.form.apiKey': 'API Key',
      'agents.form.baseUrl': 'Base URL',
      'agents.form.systemPrompt': 'Prompt',
      'confirmDialog.cancel': 'Annulla',
      'generic.saving': 'Salvataggio...',
    }
    return map[key] ?? key
  },
}))

describe('AgentFormSlideOver', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders create title when no agent provided', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByRole('heading', { name: 'Crea Agente' })).toBeInTheDocument()
  })

  it('renders edit title when agent provided', () => {
    const agent = { id: 'a1', name: 'Test Agent', model: 'gpt-4', provider: 'openai', apiKey: '', baseUrl: '', systemPrompt: '', skillIds: [] }
    render(<AgentFormSlideOver agent={agent} />)
    expect(screen.getByRole('heading', { name: 'Modifica Agente' })).toBeInTheDocument()
  })

  it('renders custom title when provided', () => {
    render(<AgentFormSlideOver title="Custom Title" />)
    expect(screen.getByRole('heading', { name: 'Custom Title' })).toBeInTheDocument()
  })

  it('renders name input', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByLabelText('Nome')).toBeInTheDocument()
  })

  it('renders provider select', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByLabelText('Provider')).toBeInTheDocument()
  })

  it('renders model input', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByLabelText('Modello')).toBeInTheDocument()
  })

  it('renders API key input', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByLabelText(/API Key/)).toBeInTheDocument()
  })

  it('renders system prompt textarea', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByLabelText('Prompt di Sistema')).toBeInTheDocument()
  })

  it('pre-fills fields in edit mode', () => {
    const agent = {
      id: 'a1',
      name: 'Existing Agent',
      model: 'claude-3',
      provider: 'anthropic',
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com',
      systemPrompt: 'You are helpful',
      skillIds: [],
    }
    render(<AgentFormSlideOver agent={agent} />)
    expect(screen.getByDisplayValue('Existing Agent')).toBeInTheDocument()
    expect(screen.getByDisplayValue('claude-3')).toBeInTheDocument()
    expect(screen.getByDisplayValue('https://api.example.com')).toBeInTheDocument()
  })

  it('renders cancel and submit buttons', () => {
    render(<AgentFormSlideOver />)
    expect(screen.getByText('Annulla')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Crea Agente' })).toBeInTheDocument()
  })

  it('shows error when submitting empty name', async () => {
    render(<AgentFormSlideOver />)
    fireEvent.click(screen.getByRole('button', { name: 'Crea Agente' }))
    await waitFor(() => {
      expect(screen.getByText('Il nome è obbligatorio')).toBeInTheDocument()
    })
  })
})
