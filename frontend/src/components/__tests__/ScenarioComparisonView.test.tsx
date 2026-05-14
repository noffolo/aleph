import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

const flatMarket = {
  id: 's3',
  name: 'Flat Market',
  confidence: 0.5,
  signals: [],
  assumptions: ['No change'],
  trend: 'neutral' as const,
  probability: 0.5,
  description: 'Sideways trading expected',
}

describe('ScenarioComparisonView', () => {
  beforeEach(() => {
    mockStoreState.scenarios = []
    mockStoreState.selectedScenarioIds = []
    mockStoreState.setSelectedScenarioIds = vi.fn()
  })

  // --- existing tests ---

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

  // --- new tests for uncovered lines ---

  it('toggles scenario on via click', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket]
    mockStoreState.selectedScenarioIds = []
    render(<ScenarioComparisonView />)
    const btn = screen.getByRole('radio', { name: 'Bull Market' })
    await userEvent.click(btn)
    expect(mockStoreState.setSelectedScenarioIds).toHaveBeenCalledWith(['s1'])
  })

  it('toggles scenario off via click', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket]
    mockStoreState.selectedScenarioIds = ['s1', 's2']
    render(<ScenarioComparisonView />)
    const btn = screen.getByRole('radio', { name: 'Bull Market' })
    await userEvent.click(btn)
    expect(mockStoreState.setSelectedScenarioIds).toHaveBeenCalledWith(['s2'])
  })

  it('enforces max 3 scenarios limit', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket, flatMarket, {
      id: 's4', name: 'Extra', confidence: 0.5, signals: [], assumptions: [],
      trend: 'neutral' as const, probability: 0.5,
    }]
    mockStoreState.selectedScenarioIds = ['s1', 's2', 's3']
    render(<ScenarioComparisonView />)
    const extraBtn = screen.getByRole('radio', { name: 'Extra' })
    await userEvent.click(extraBtn)
    expect(mockStoreState.setSelectedScenarioIds).not.toHaveBeenCalled()
  })

  it('renders neutral trend for flat scenario', () => {
    mockStoreState.scenarios = [flatMarket]
    mockStoreState.selectedScenarioIds = ['s3']
    render(<ScenarioComparisonView />)
    expect(screen.getByText('neutral')).toBeInTheDocument()
  })

  it('renders description field when scenario has description', () => {
    mockStoreState.scenarios = [flatMarket]
    mockStoreState.selectedScenarioIds = ['s3']
    render(<ScenarioComparisonView />)
    expect(screen.getByText(/Sideways trading expected/)).toBeInTheDocument()
  })

  it('handles ArrowRight keyboard navigation in scenario selector', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket]
    render(<ScenarioComparisonView />)
    const buttons = screen.getAllByRole('radio')
    buttons[0].focus()
    fireEvent.keyDown(buttons[0].parentElement!, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(buttons[1])
  })

  it('handles ArrowDown keyboard navigation', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket, flatMarket]
    render(<ScenarioComparisonView />)
    const buttons = screen.getAllByRole('radio')
    buttons[0].focus()
    fireEvent.keyDown(buttons[0].parentElement!, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(buttons[1])
  })

  it('handles ArrowLeft keyboard navigation', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket]
    render(<ScenarioComparisonView />)
    const buttons = screen.getAllByRole('radio')
    buttons[1].focus()
    fireEvent.keyDown(buttons[1].parentElement!, { key: 'ArrowLeft' })
    expect(document.activeElement).toBe(buttons[0])
  })

  it('handles ArrowUp keyboard navigation', async () => {
    mockStoreState.scenarios = [bullMarket, bearMarket, flatMarket]
    render(<ScenarioComparisonView />)
    const buttons = screen.getAllByRole('radio')
    buttons[1].focus()
    fireEvent.keyDown(buttons[1].parentElement!, { key: 'ArrowUp' })
    expect(document.activeElement).toBe(buttons[0])
  })

  it('renders scenario card with correct aria-checked attribute', () => {
    mockStoreState.scenarios = [bullMarket]
    mockStoreState.selectedScenarioIds = ['s1']
    render(<ScenarioComparisonView />)
    const btn = screen.getByRole('radio', { name: 'Bull Market' })
    expect(btn.getAttribute('aria-checked')).toBe('true')
  })
})
