import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { OracleView } from '../OracleView'

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      const state: Record<string, unknown> = {
        projectID: '',
        predictions: [],
        setPredictions: vi.fn(),
        setLastError: vi.fn(),
        expandedSections: { 'oracle.predictions': true, 'oracle.sentiment': true },
        toggleSection: vi.fn(),
      }
      if (typeof selector === 'function') return selector(state)
      return state
    }),
    { getState: vi.fn(() => ({ setLastError: vi.fn() })) },
  ),
}))

vi.mock('../../api/factory', () => ({
  nlpClient: {
    streamPredictions: async function* () {},
    recordFeedback: vi.fn().mockResolvedValue(undefined),
    analyzeSentiment: vi.fn(),
  },
}))

vi.mock('../../lib/errorReporter', () => ({
  reportError: vi.fn(),
}))

vi.mock('../../i18n', () => ({
  t: (key: string, _opts?: Record<string, unknown>) => {
    const map: Record<string, string> = {
      'oracle.title': 'Oracolo',
      'oracle.empty': 'Nessuna previsione',
      'oracle.sentimentTitle': 'Sentiment',
      'oracle.sentimentSubtitle': 'Analisi',
      'oracle.sentimentPlaceholder': 'Scrivi testo...',
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
  Brain: () => null,
  TrendingUp: () => null,
  BarChart3: () => null,
  Settings2: () => null,
  Activity: () => null,
  ChevronDown: () => null,
  Zap: () => null,
  AlertTriangle: () => null,
  Clock: () => null,
  ThumbsUp: () => null,
  ThumbsDown: () => null,
  MessageSquareText: () => null,
}))

describe('OracleView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders title', () => {
    render(<OracleView />)
    // Title appears in multiple GlassPanel headers
    expect(screen.getAllByText('Oracolo').length).toBeGreaterThan(0)
  })

  it('renders empty state when no predictions', () => {
    render(<OracleView />)
    expect(screen.getByText('Nessuna previsione')).toBeInTheDocument()
  })

  it('renders skeleton when isLoading prop is true', () => {
    render(<OracleView isLoading={true} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error when provided', () => {
    render(<OracleView error="Fetch error" />)
    expect(screen.getByTestId('inline-error')).toBeInTheDocument()
  })

  it('renders sentiment analysis section', () => {
    render(<OracleView />)
    expect(screen.getByText('Sentiment')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Scrivi testo...')).toBeInTheDocument()
  })
})
