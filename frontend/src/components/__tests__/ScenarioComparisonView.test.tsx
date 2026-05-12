import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ScenarioComparisonView } from '../../views/ScenarioComparisonView'

const mockStoreState = {
  scenarios: [] as any[],
  selectedScenarioIds: [] as string[],
  setSelectedScenarioIds: vi.fn(),
}

vi.mock('../../store/useStore', () => ({
  useStore: (selector?: (state: unknown) => unknown) => {
    if (typeof selector === 'function') return selector(mockStoreState as any)
    return mockStoreState
  },
}))

const bullMarket = {
  id: 's1',
  name: 'Bull Market',
  confidence: 0.85,
  signals: [{ id: 'sig1', name: 'Momentum', strength: 0.8 }],
  assumptions: ['Growth continues'],
  trend: 'up' as const,
  probability: 0.75,
}

const bearMarket = {
  id: 's2',
  name: 'Bear Market',
  confidence: 0.65,
  signals: [{ id: 'sig2', name: 'Recession', strength: 0.7 }],
  assumptions: ['High inflation'],
  trend: 'down' as const,
  probability: 0.35,
}

describe('ScenarioComparisonView', () => {
  beforeEach(() => {
    mockStoreState.scenarios = []
    mockStoreState.selectedScenarioIds = []
    mockStoreState.setSelectedScenarioIds = vi.fn()
  })

  it('renders empty state when no scenarios', () => {
    render(<ScenarioComparisonView />)
    expect(screen.getByText('No Scenarios Found')).toBeInTheDocument()
  })

  it('renders scenario selector buttons', () => {
    mockStoreState.scenarios = [bullMarket, bearMarket]
    render(<ScenarioComparisonView />)
    expect(screen.getByText('Bull Market')).toBeInTheDocument()
    expect(screen.getByText('Bear Market')).toBeInTheDocument()
  })

  it('shows "Select 2 or 3 scenarios" message when none selected', () => {
    mockStoreState.scenarios = [{ ...bullMarket }]
    mockStoreState.selectedScenarioIds = []
    render(<ScenarioComparisonView />)
    expect(screen.getByText(/Select 2 or 3 scenarios/)).toBeInTheDocument()
  })

  it('renders comparison cards when scenarios selected', () => {
    mockStoreState.scenarios = [{ ...bullMarket }]
    mockStoreState.selectedScenarioIds = ['s1']
    render(<ScenarioComparisonView />)
    expect(screen.getAllByText('Bull Market')).toHaveLength(2)
    expect(screen.getByText(/85.0%/)).toBeInTheDocument()
  })

  it('renders signals section when scenario selected', () => {
    mockStoreState.scenarios = [{ ...bullMarket }]
    mockStoreState.selectedScenarioIds = ['s1']
    render(<ScenarioComparisonView />)
    expect(screen.getByText('Momentum')).toBeInTheDocument()
  })

  it('renders assumptions section', () => {
    mockStoreState.scenarios = [{ ...bullMarket, description: 'Optimistic scenario' }]
    mockStoreState.selectedScenarioIds = ['s1']
    render(<ScenarioComparisonView />)
    expect(screen.getByText('Growth continues')).toBeInTheDocument()
    expect(screen.getByText(/Optimistic scenario/)).toBeInTheDocument()
  })

  it('renders correct trend for up scenario', () => {
    mockStoreState.scenarios = [{ ...bullMarket }]
    mockStoreState.selectedScenarioIds = ['s1']
    render(<ScenarioComparisonView />)
    expect(screen.getByText('up')).toBeInTheDocument()
  })

  it('renders correct trend for down scenario', () => {
    mockStoreState.scenarios = [{ ...bearMarket }]
    mockStoreState.selectedScenarioIds = ['s2']
    render(<ScenarioComparisonView />)
    expect(screen.getByText('down')).toBeInTheDocument()
  })

  it('renders probability footer', () => {
    mockStoreState.scenarios = [{ ...bullMarket }]
    mockStoreState.selectedScenarioIds = ['s1']
    render(<ScenarioComparisonView />)
    expect(screen.getByText('75.0%')).toBeInTheDocument()
  })
})
