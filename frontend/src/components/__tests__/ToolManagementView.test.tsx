import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

const { mockSetSlideOverContent, mockSetSearchQuery, mockStore } = vi.hoisted(() => {
  const mockSetSlideOverContent = vi.fn()
  const mockSetSearchQuery = vi.fn()
  const mockStore = () => ({
    tools: [
      { id: 't1', name: 'CSV Parser', description: 'Parse CSV files', healthStatus: 'healthy', version: '1.2.0', category: 'data', lastCheckedAt: '2026-01-01T00:00:00Z' },
      { id: 't2', name: 'Sentiment Analyzer', description: 'Analyze sentiment', healthStatus: 'warning', version: '0.9.0', category: 'nlp', lastCheckedAt: null },
      { id: 't3', name: 'Data Exporter', description: null, healthStatus: 'error', version: null, category: null, lastCheckedAt: '2025-06-15T12:30:00Z' },
    ],
    setSlideOverContent: mockSetSlideOverContent,
  })
  return { mockSetSlideOverContent, mockSetSearchQuery, mockStore }
})

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((sel: (s: ReturnType<typeof mockStore>) => unknown) => sel(mockStore())),
    { getState: () => ({ ...mockStore(), setSlideOverContent: mockSetSlideOverContent }) },
  ),
}))

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['', mockSetSearchQuery]),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => key,
}))

vi.mock('lucide-react', () => {
  const Icon = (name: string) => {
    const Comp = (props: React.SVGProps<SVGSVGElement>) => <svg {...props} data-testid={`icon-${name}`} />
    Comp.displayName = name
    return Comp
  }
  return {
    Terminal: Icon('Terminal'),
    Plus: Icon('Plus'),
    Trash2: Icon('Trash2'),
    Activity: Icon('Activity'),
    AlertCircle: Icon('AlertCircle'),
    CheckCircle2: Icon('CheckCircle2'),
    Search: Icon('Search'),
    BarChart3: Icon('BarChart3'),
  }
})

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
}))

vi.mock('../ui/InlineError', () => ({
  InlineError: ({ message }: { message: string }) => <div data-testid="inline-error">{message}</div>,
}))

import { ToolManagementView } from '../ToolManagementView'

describe('ToolManagementView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // --- Render states ---
  it('renders loading skeleton when isLoading is true', () => {
    render(<ToolManagementView isLoading={true} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error message when error is provided', () => {
    render(<ToolManagementView error="Network error" />)
    expect(screen.getByTestId('inline-error')).toHaveTextContent('Network error')
  })

  it('renders header with Italian title', () => {
    render(<ToolManagementView />)
    expect(screen.getByText('Gestione Strumenti')).toBeInTheDocument()
  })

  // --- Tool rendering ---
  it('renders all tools from store', () => {
    render(<ToolManagementView />)
    expect(screen.getByText('CSV Parser')).toBeInTheDocument()
    expect(screen.getByText('Sentiment Analyzer')).toBeInTheDocument()
    expect(screen.getByText('Data Exporter')).toBeInTheDocument()
  })

  it('renders tool description or fallback', () => {
    render(<ToolManagementView />)
    expect(screen.getByText('Parse CSV files')).toBeInTheDocument()
    expect(screen.getByText('Nessuna descrizione fornita.')).toBeInTheDocument()
  })

  it('renders tool version and category info', () => {
    render(<ToolManagementView />)
    expect(screen.getByText('1.2.0')).toBeInTheDocument()
    expect(screen.getByText('0.9.0')).toBeInTheDocument()
    expect(screen.getByText('data')).toBeInTheDocument()
    expect(screen.getByText('nlp')).toBeInTheDocument()
  })

  // --- Empty state ---
  it('shows "nessun strumento" when no tools match filter', async () => {
    const { useQueryState } = await import('nuqs')
    ;(useQueryState as ReturnType<typeof vi.fn>).mockReturnValueOnce(['nonexistent', mockSetSearchQuery])
    render(<ToolManagementView />)
    expect(screen.getByText('Nessun strumento trovato.')).toBeInTheDocument()
  })

  // --- Search input ---
  it('renders search input with translated placeholder', () => {
    render(<ToolManagementView />)
    const input = screen.getByPlaceholderText('tools.search')
    expect(input).toBeInTheDocument()
  })

  it('calls setSearchQuery on input change', () => {
    render(<ToolManagementView />)
    const input = screen.getByPlaceholderText('tools.search')
    fireEvent.change(input, { target: { value: 'CSV' } })
    expect(mockSetSearchQuery).toHaveBeenCalledWith('CSV')
  })

  // --- Status icons ---
  it('shows health status labels for tools', () => {
    render(<ToolManagementView />)
    // Renders tool health status as text with CSS uppercase
    const toolContainer = screen.getByText('CSV Parser').closest('div')
    expect(toolContainer?.textContent).toMatch(/healthy/i)
    expect(screen.getByText('CSV Parser')).toBeInTheDocument()
  })

  // --- Intelligence button ---
  it('opens tool intelligence slideover on button click', () => {
    render(<ToolManagementView />)
    fireEvent.click(screen.getByText('Intelligence'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'tool-intelligence',
      title: 'Tool Intelligence',
    })
  })

  // --- Detail button ---
  it('opens detail slideover with tool data on click', () => {
    render(<ToolManagementView />)
    const detailButtons = screen.getAllByRole('button', { name: /Dettagli/ })
    expect(detailButtons.length).toBeGreaterThan(0)
    fireEvent.click(detailButtons[0])
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'tool',
      title: 'Dettagli Tool',
      data: expect.objectContaining({ id: 't1', name: 'CSV Parser' }),
    })
  })

  // --- Inline mode ---
  it('renders with inline styling when inline prop is true', () => {
    const { container } = render(<ToolManagementView inline={true} />)
    const root = container.firstElementChild as HTMLElement
    expect(root.className).toContain('p-6')
    expect(root.className).not.toContain('max-w-6xl')
  })
})
