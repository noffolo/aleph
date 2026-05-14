import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { OracleView } from '../OracleView'

const { getPredictions, setPredictions, getMockStoreState } = vi.hoisted(() => {
  let predictions: any[] = []
  return {
    getPredictions: () => predictions,
    setPredictions: (p: any[]) => { predictions = p },
    getMockStoreState: () => ({
      projectID: '',
      predictions,
      setPredictions: vi.fn((p: any[]) => { predictions = p }),
      setLastError: vi.fn(),
      expandedSections: { 'oracle.predictions': true, 'oracle.sentiment': true },
      toggleSection: vi.fn(),
    }),
  }
})

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      const state = getMockStoreState()
      if (typeof selector === 'function') return selector(state)
      return state
    }),
    { getState: vi.fn(() => ({ setLastError: vi.fn() })) },
  ),
}))

const { mockRecordFeedback, mockAnalyzeSentiment } = vi.hoisted(() => ({
  mockRecordFeedback: vi.fn().mockResolvedValue(undefined),
  mockAnalyzeSentiment: vi.fn(),
}))

vi.mock('../../api/factory', () => ({
  nlpClient: {
    streamPredictions: async function* () {},
    recordFeedback: (...args: any[]) => mockRecordFeedback(...args),
    analyzeSentiment: (...args: any[]) => mockAnalyzeSentiment(...args),
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
    setPredictions([])
  })

  it('renders title', () => {
    render(<OracleView />)
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

  it('renders prediction when predictions exist in store', () => {
    setPredictions([{ entityId: 'BTC', probability: 0.85, predictedState: 'ACTION_REQUIRED', explanation: 'Trend bullish' }])
    render(<OracleView />)
    expect(screen.getByText('BTC')).toBeInTheDocument()
  })

  it('renders high confidence text', () => {
    setPredictions([{ entityId: 'ETH', probability: 0.92, predictedState: 'STABLE', explanation: 'Steady growth' }])
    render(<OracleView />)
    expect(screen.getByText('92%')).toBeInTheDocument()
  })

  it('renders medium confidence text', () => {
    setPredictions([{ entityId: 'SOL', probability: 0.65, predictedState: 'STABLE', explanation: 'Moderate' }])
    render(<OracleView />)
    expect(screen.getByText('65%')).toBeInTheDocument()
  })

  it('renders low confidence text', () => {
    setPredictions([{ entityId: 'DOGE', probability: 0.25, predictedState: 'STABLE', explanation: 'Risky' }])
    render(<OracleView />)
    expect(screen.getByText('25%')).toBeInTheDocument()
  })

  it('renders feedback buttons', () => {
    setPredictions([{ entityId: 'BTC', probability: 0.85, predictedState: 'UP', explanation: 'Trend bullish' }])
    render(<OracleView />)
    expect(screen.getByTitle('oracle.correctPrediction')).toBeInTheDocument()
    expect(screen.getByTitle('oracle.wrongPrediction')).toBeInTheDocument()
  })

  it('renders execute button disabled when sentiment text is empty', () => {
    render(<OracleView />)
    expect(screen.getByText('generic.execute')).toBeDisabled()
  })

  it('enables execute button when sentiment text has content', () => {
    render(<OracleView />)
    fireEvent.change(screen.getByPlaceholderText('Scrivi testo...'), { target: { value: 'Il mercato sta crollando' } })
    expect(screen.getByText('generic.execute')).not.toBeDisabled()
  })

  it('executes sentiment analysis and renders positive result', async () => {
    mockAnalyzeSentiment.mockResolvedValue({ score: 0.88, label: 'positive' })
    render(<OracleView />)
    fireEvent.change(screen.getByPlaceholderText('Scrivi testo...'), { target: { value: 'Il mercato sta salendo' } })
    await userEvent.click(screen.getByText('generic.execute'))
    await waitFor(() => {
      expect(screen.getByText(/88%/)).toBeInTheDocument()
    })
  })

  it('executes sentiment analysis and renders negative result', async () => {
    mockAnalyzeSentiment.mockResolvedValue({ score: 0.12, label: 'negative' })
    render(<OracleView />)
    fireEvent.change(screen.getByPlaceholderText('Scrivi testo...'), { target: { value: 'Il mercato crolla' } })
    await userEvent.click(screen.getByText('generic.execute'))
    await waitFor(() => {
      expect(screen.getByText(/12%/)).toBeInTheDocument()
    })
  })

  it('handles sentiment analysis error gracefully', async () => {
    mockAnalyzeSentiment.mockRejectedValue(new Error('Network error'))
    render(<OracleView />)
    fireEvent.change(screen.getByPlaceholderText('Scrivi testo...'), { target: { value: 'Test' } })
    await userEvent.click(screen.getByText('generic.execute'))
    await waitFor(() => {
      expect(screen.getByText('errors.analysis')).toBeInTheDocument()
    })
  })

  it('does not execute sentiment when text is empty', async () => {
    render(<OracleView />)
    const btn = screen.getByText('generic.execute')
    expect(btn).toBeDisabled()
  })

  it('sends correct feedback and disables thumbs-up when already given', async () => {
    setPredictions([{ entityId: 'BTC', probability: 0.85, predictedState: 'UP', explanation: 'Trend bullish' }])
    render(<OracleView />)
    const thumbsUp = screen.getByTitle('oracle.correctPrediction')
    await userEvent.click(thumbsUp)
    expect(mockRecordFeedback).toHaveBeenCalledWith({ entityId: 'BTC', isCorrect: true, feedbackType: 'prediction' })
  })

  it('handles feedback failure by resetting button state', async () => {
    mockRecordFeedback.mockRejectedValueOnce(new Error('network error'))
    setPredictions([{ entityId: 'BTC', probability: 0.85, predictedState: 'UP', explanation: 'Trend bullish' }])
    render(<OracleView />)
    const thumbsUp = screen.getByTitle('oracle.correctPrediction')
    await userEvent.click(thumbsUp)
    await waitFor(() => {
      expect(mockRecordFeedback).toHaveBeenCalled()
    })
  })

  it('renders multiple predictions in grid', () => {
    setPredictions([
      { entityId: 'BTC', probability: 0.85, predictedState: 'STABLE', explanation: 'Bullish trend' },
      { entityId: 'ETH', probability: 0.65, predictedState: 'STABLE', explanation: 'Moderate growth' },
    ])
    render(<OracleView />)
    expect(screen.getByText('BTC')).toBeInTheDocument()
    expect(screen.getByText('ETH')).toBeInTheDocument()
  })

  it('toggles advanced settings button', async () => {
    render(<OracleView />)
    const settingsBtn = screen.getByTitle('oracle.advancedSettings')
    await userEvent.click(settingsBtn)
  })

  it('renders inline mode without max-w wrapper', () => {
    setPredictions([])
    render(<OracleView inline={true} />)
    expect(screen.getByText('Nessuna previsione')).toBeInTheDocument()
  })
})
