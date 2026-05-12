import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ToolIntelligenceView from '../ToolIntelligenceView'

vi.mock('../../api/client', () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiPatch: vi.fn(),
}))

vi.mock('../../lib/errorReporter', () => ({
  reportError: vi.fn(),
}))

vi.mock('../../store/useStore', () => ({
  useStore: vi.fn((selector?: (state: unknown) => unknown) => {
    const state = { projectID: 'proj-1' }
    if (typeof selector === 'function') return selector(state)
    return state
  }),
}))

vi.mock('lucide-react', () => ({
  Activity: () => null,
  Users: () => null,
  Shield: () => null,
  Star: () => null,
  AlertTriangle: () => null,
  CheckCircle2: () => null,
  BarChart3: () => null,
}))

import { apiGet } from '../../api/client'
const mockApiGet = apiGet as ReturnType<typeof vi.fn>

describe('ToolIntelligenceView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state initially', () => {
    mockApiGet.mockReturnValue(new Promise(() => {}))
    render(<ToolIntelligenceView />)
    expect(screen.getByText('Loading intelligence data...')).toBeInTheDocument()
  })

  it('renders empty state when no tools', async () => {
    mockApiGet.mockResolvedValueOnce([])
    render(<ToolIntelligenceView />)
    await waitFor(() => {
      expect(screen.getByText('No tool data available')).toBeInTheDocument()
    })
  })

  it('renders tool intelligence cards after load', async () => {
    mockApiGet.mockResolvedValueOnce([
      {
        id: 't1',
        name: 'Tool1',
        execCount: 150,
        avgDuration: 45,
        riskScore: 25,
        usageFreq: 'high',
        topUsers: ['user1'],
        recommendations: ['Improve speed'],
        anomalies: [],
        relatedTools: ['t2'],
        warnings: [],
      },
    ])
    render(<ToolIntelligenceView />)
    await waitFor(() => {
      expect(screen.getByText('CodeFlow Analysis')).toBeInTheDocument()
      expect(screen.getByText('Usage Patterns')).toBeInTheDocument()
      expect(screen.getByText('Security Intelligence')).toBeInTheDocument()
      expect(screen.getByText('Cross-Context Recommendations')).toBeInTheDocument()
    })
  })

  it('renders tool names in cards', async () => {
    mockApiGet.mockResolvedValueOnce([
      {
        id: 't1',
        name: 'Analyzer',
        execCount: 100,
        avgDuration: 30,
        riskScore: 15,
        usageFreq: 'medium',
        topUsers: [],
        recommendations: [],
        anomalies: [],
        relatedTools: [],
        warnings: [],
      },
    ])
    render(<ToolIntelligenceView />)
    await waitFor(() => {
      const elements = screen.getAllByText('Analyzer')
      expect(elements.length).toBeGreaterThanOrEqual(1)
    })
  })
})
