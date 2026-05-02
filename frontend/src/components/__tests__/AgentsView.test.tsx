import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AgentsView } from '../AgentsView'
import type { Agent } from '../../store/types'

// --- Mocks ---

const mockSetSlideOverContent = vi.fn()
const mockSetAgents = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        const state = {
          setAgents: mockSetAgents,
          selectedObject: 'proj-1',
          setSlideOverContent: mockSetSlideOverContent,
          agents: [],
        }
        return selector(state)
      }
      return {
        setAgents: mockSetAgents,
        selectedObject: 'proj-1',
        setSlideOverContent: mockSetSlideOverContent,
        agents: [],
      }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({ setSlideOverContent: mockSetSlideOverContent })),
    },
  ),
}))

vi.mock('../../hooks/useCursorPagination', () => ({
  useCursorPagination: vi.fn(({ initialItems }: { initialItems: Agent[] }) => ({
    items: initialItems,
    hasMore: false,
    loadMore: vi.fn(),
    loading: false,
  })),
}))

// Mock nuqs
vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['', vi.fn()]),
}))

// Mock i18n
vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'agents.title': 'Gestore Agenti',
      'agents.subtitle': 'Configura agenti AI con qualsiasi provider',
      'agents.create': 'Nuovo Agente',
      'agents.edit': 'Modifica Agente',
      'agents.search': 'Cerca...',
      'agents.noSystemPrompt': 'Nessun prompt di sistema configurato.',
      'generic.serviceActive': 'Servizio Attivo',
      'generic.offline': 'Offline',
      'generic.noAgents': 'Nessun agente configurato',
      'generic.loadMore': 'Carica Altri',
      'generic.loadingLower': 'Caricamento...',
      'generic.loading': 'Caricamento...',
    }
    return map[key] ?? key
  },
}))

// --- Test helpers ---

function makeAgent(id: string, overrides?: Partial<Agent>): Agent {
  return {
    id,
    name: `Agent ${id}`,
    model: 'gpt-4',
    systemPrompt: 'You are helpful.',
    provider: 'openai',
    apiKey: '',
    baseUrl: '',
    ...overrides,
  }
}

// --- Tests ---

describe('AgentsView', () => {
  const mockOnCreate = vi.fn()
  const mockOnDelete = vi.fn()
  const mockOnUpdate = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  // —— Rendering ——

  it('renders the title and subtitle', () => {
    render(
      <AgentsView
        agents={[]}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByText('Gestore Agenti')).toBeInTheDocument()
    expect(screen.getByText('Configura agenti AI con qualsiasi provider')).toBeInTheDocument()
  })

  it('renders agent cards when agents are provided', () => {
    const agents = [makeAgent('1'), makeAgent('2')]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByText('Agent 1')).toBeInTheDocument()
    expect(screen.getByText('Agent 2')).toBeInTheDocument()
  })

  it('renders agent model and provider badges', () => {
    const agents = [makeAgent('1', { model: 'gpt-4', provider: 'openai' })]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByText('gpt-4')).toBeInTheDocument()
    expect(screen.getByText('openai')).toBeInTheDocument()
  })

  it('displays system prompt when present', () => {
    const agents = [makeAgent('1', { systemPrompt: 'Be concise.' })]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByText('Be concise.')).toBeInTheDocument()
  })

  it('displays fallback text when systemPrompt is missing', () => {
    const agents = [makeAgent('1', { systemPrompt: '' })]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByText('Nessun prompt di sistema configurato.')).toBeInTheDocument()
  })

  it('shows service active indicator when ollamaHealthy=true', () => {
    const agents = [makeAgent('1')]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
        ollamaHealthy={true}
      />,
    )
    expect(screen.getByText('Servizio Attivo')).toBeInTheDocument()
  })

  it('shows offline indicator when ollamaHealthy=false', () => {
    const agents = [makeAgent('1')]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
        ollamaHealthy={false}
      />,
    )
    expect(screen.getByText('Offline')).toBeInTheDocument()
  })

  // —— Empty state ——

  it('renders empty state when no agents', () => {
    render(
      <AgentsView
        agents={[]}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByText('Nessun agente configurato')).toBeInTheDocument()
  })

  // —— Loading state ——

  it('renders skeleton loader when isLoading is true', () => {
    render(
      <AgentsView
        agents={[]}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
        isLoading={true}
      />,
    )
    // SkeletonLoader renders a div with animate-pulse
    const pulseEls = document.querySelectorAll('.animate-pulse')
    expect(pulseEls.length).toBeGreaterThan(0)
  })

  // —— Error state ——

  it('renders error message when error is provided', () => {
    render(
      <AgentsView
        agents={[]}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
        error="Something went wrong"
      />,
    )
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  // —— Interactions ——

  it('opens slide over on create button click', () => {
    const agents = [makeAgent('1')]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    fireEvent.click(screen.getByLabelText('Create new agent'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'agent-form',
      title: 'Nuovo Agente',
      data: undefined,
    })
  })

  // —— Inline mode ——

  it('does not apply max-w-6xl when inline=true', () => {
    const { container } = render(
      <AgentsView
        agents={[]}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
        inline={true}
      />,
    )
    const region = container.querySelector('[role="region"]')
    expect(region?.className).not.toContain('max-w-6xl')
  })

  // —— Accessibility ——

  it('has region role with aria-label', () => {
    render(
      <AgentsView
        agents={[]}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByRole('region', { name: 'Agents' })).toBeInTheDocument()
  })

  it('edit/delete buttons have aria-labels', () => {
    const agents = [makeAgent('1')]
    render(
      <AgentsView
        agents={agents}
        onCreateAgent={mockOnCreate}
        onDeleteAgent={mockOnDelete}
        onUpdateAgent={mockOnUpdate}
      />,
    )
    expect(screen.getByLabelText('Edit agent Agent 1')).toBeInTheDocument()
    expect(screen.getByLabelText('Delete agent Agent 1')).toBeInTheDocument()
  })
})
