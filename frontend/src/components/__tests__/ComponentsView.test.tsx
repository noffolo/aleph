import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ComponentsView } from '../ComponentsView'

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      const state = {
        setSlideOverContent: mockSetSlideOverContent,
        expandedSections: { 'components.list': true },
        toggleSection: vi.fn(),
      }
      if (typeof selector === 'function') return selector(state)
      return state
    }),
    { getState: vi.fn(() => ({ setSlideOverContent: mockSetSlideOverContent })) },
  ),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'components.title': 'Catalogo Componenti',
      'components.subtitle': 'Gestisci componenti',
      'components.register': 'Registra',
      'components.search': 'Cerca componenti...',
      'components.edit': 'Modifica Componente',
      'components.betaScore': '(beta)',
    }
    return map[key] ?? key
  },
}))

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">loading</div>,
}))

vi.mock('../ui/InlineError', () => ({
  InlineError: ({ message }: { message: string }) => <div data-testid="inline-error">{message}</div>,
}))

vi.mock('lucide-react', () => ({
  Cpu: () => null,
  Search: () => null,
  Zap: () => null,
  ToggleLeft: () => null,
  Plus: () => null,
  Eye: () => null,
  ChevronDown: () => null,
}))

const mockSetSlideOverContent = vi.fn()

describe('ComponentsView', () => {
  const mockOnUpdateComponentStatus = vi.fn()
  const mockOnRegisterComponent = vi.fn()
  const mockOnGetComponent = vi.fn()

  const components = [
    {
      id: 'c1',
      name: 'Analyzer',
      description: 'Analyzes data',
      version: '1.0',
      type: 'tool',
      category: 'analytical',
      source: 'registry',
      status: 'active',
      approvalStatus: 'approved',
    },
    {
      id: 'c2',
      name: 'Processor',
      description: 'Processes events',
      version: '2.1',
      type: 'pipeline',
      category: 'transformative',
      source: 'user',
      status: 'paused',
      approvalStatus: 'review',
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders title and subtitle', () => {
    render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText('Catalogo Componenti')).toBeInTheDocument()
  })

  it('renders component cards', () => {
    render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText('Analyzer')).toBeInTheDocument()
    expect(screen.getByText('Processor')).toBeInTheDocument()
  })

  it('renders component descriptions', () => {
    render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText('Analyzes data')).toBeInTheDocument()
    expect(screen.getByText('Processes events')).toBeInTheDocument()
  })

  it('renders empty state when no components', () => {
    render(
      <ComponentsView
        components={[]}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText('Nessun componente registrato nel catalogo')).toBeInTheDocument()
  })

  it('filters components by search query', () => {
    render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    const searchInput = screen.getByPlaceholderText('Cerca componenti...')
    fireEvent.change(searchInput, { target: { value: 'Analyzer' } })
    expect(screen.getByText('Analyzer')).toBeInTheDocument()
    expect(screen.queryByText('Processor')).not.toBeInTheDocument()
  })

  it('shows no match message on filter miss', () => {
    render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    const searchInput = screen.getByPlaceholderText('Cerca componenti...')
    fireEvent.change(searchInput, { target: { value: 'zzz-nonexistent' } })
    expect(screen.getByText('Nessun componente corrisponde al filtro')).toBeInTheDocument()
  })

  it('renders skeleton when loading', () => {
    render(
      <ComponentsView
        components={[]}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
        isLoading={true}
      />,
    )
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error when provided', () => {
    render(
      <ComponentsView
        components={[]}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
        error="Fetch error"
      />,
    )
    expect(screen.getByTestId('inline-error')).toBeInTheDocument()
  })

  it('calls onUpdateComponentStatus on activate button', () => {
    render(
      <ComponentsView
        components={[components[1]]}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    fireEvent.click(screen.getByText('Attiva'))
    expect(mockOnUpdateComponentStatus).toHaveBeenCalledWith('c2', 'active')
  })

  it('calls onUpdateComponentStatus on pause button', () => {
    render(
      <ComponentsView
        components={[components[0]]}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    fireEvent.click(screen.getByText('Pausa'))
    expect(mockOnUpdateComponentStatus).toHaveBeenCalledWith('c1', 'paused')
  })

  it('renders execution command when present on component', () => {
    const comps = [{ ...components[0], executionCommand: 'npm run analyze' }]
    render(
      <ComponentsView
        components={comps}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText(/\$ npm run analyze/)).toBeInTheDocument()
  })

  it('renders component detail slide over on Dettagli click', () => {
    render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    fireEvent.click(screen.getByLabelText('View component Analyzer details'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'component-detail',
      title: 'Analyzer',
      data: { componentId: 'c1' },
    })
  })

  it('renders approval status badge when not approved', () => {
    const comps = [{ ...components[0], approvalStatus: 'pending' }]
    render(
      <ComponentsView
        components={comps}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText('pending')).toBeInTheDocument()
  })

  it('renders status badges for different health states', () => {
    const comps = [
      { ...components[0], status: 'running' },
      { ...components[0], id: 'c3', name: 'Watcher', status: 'failed' },
    ]
    render(
      <ComponentsView
        components={comps}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText('running')).toBeInTheDocument()
    expect(screen.getByText('failed')).toBeInTheDocument()
  })

  it('renders latency, trust, brier, cpu, memory metadata when present', () => {
    const comps = [
      {
        ...components[0],
        avgLatencyMs: 42,
        avgExecTimeMs: 150,
        trustScore: 0.85,
        avgBrierScore: 0.123,
        avgCpuUsage: 12.5,
        avgMemoryMb: 256,
      },
    ]
    render(
      <ComponentsView
        components={comps}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
      />,
    )
    expect(screen.getByText(/42ms/)).toBeInTheDocument()
    expect(screen.getByText(/Esecuzione 150ms/)).toBeInTheDocument()
    expect(screen.getByText(/Trust \(beta\) 0.85/)).toBeInTheDocument()
    expect(screen.getByText(/Brier \(beta\) 0.123/)).toBeInTheDocument()
    expect(screen.getByText(/CPU 12.5%/)).toBeInTheDocument()
    expect(screen.getByText(/256MB/)).toBeInTheDocument()
  })

  it('renders inline mode without max-w wrapper', () => {
    const { container } = render(
      <ComponentsView
        components={components}
        onUpdateComponentStatus={mockOnUpdateComponentStatus}
        onRegisterComponent={mockOnRegisterComponent}
        onGetComponent={mockOnGetComponent}
        inline={true}
      />,
    )
    const root = container.firstElementChild as HTMLElement
    expect(root.className).not.toContain('max-w-6xl')
  })
})
