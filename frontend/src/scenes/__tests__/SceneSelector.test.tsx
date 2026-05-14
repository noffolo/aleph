import React, { Suspense } from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SceneSelector } from '../SceneSelector'

vi.mock('../../components/terminal/TerminalView', () => ({
  TerminalView: () => <div data-testid="terminal-view">Terminal View</div>,
}))

vi.mock('../../components/DashboardView', () => ({
  DashboardView: () => <div data-testid="dashboard-view">Dashboard</div>,
}))

vi.mock('../../components/SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading</div>,
}))

vi.mock('../ExploreScene', () => ({ ExploreScene: () => <div data-testid="explore-scene">Explore</div> }))
vi.mock('../AgentsScene', () => ({ AgentsScene: () => <div data-testid="agents-scene">Agents</div> }))
vi.mock('../SystemScene', () => ({ SystemScene: () => <div data-testid="system-scene">System</div> }))

const mockUseStore = vi.hoisted(() => vi.fn())

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    mockUseStore,
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({ slideOverContent: null })),
    },
  ),
}))

describe('SceneSelector', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const wrapWithSuspense = (children: React.ReactNode) => (
    <Suspense fallback={<div>Loading...</div>}>{children}</Suspense>
  )

  it('renders TerminalView when scene is null (default)', () => {
    mockUseStore.mockImplementation((selector?: (s: unknown) => unknown) => {
      if (typeof selector === 'function') return selector({ currentScene: null })
      return { currentScene: null }
    })
    render(wrapWithSuspense(<SceneSelector />))
    expect(screen.getByTestId('terminal-view')).toBeDefined()
  })

  it('renders ExploreScene when scene is explore', async () => {
    mockUseStore.mockImplementation((selector?: (s: unknown) => unknown) => {
      if (typeof selector === 'function') return selector({ currentScene: 'explore' })
      return { currentScene: 'explore' }
    })
    render(wrapWithSuspense(<SceneSelector />))
    expect(await screen.findByTestId('explore-scene')).toBeDefined()
  })

  it('renders AgentsScene when scene is agents', async () => {
    mockUseStore.mockImplementation((selector?: (s: unknown) => unknown) => {
      if (typeof selector === 'function') return selector({ currentScene: 'agents' })
      return { currentScene: 'agents' }
    })
    render(wrapWithSuspense(<SceneSelector />))
    expect(await screen.findByTestId('agents-scene')).toBeDefined()
  })

  it('renders SystemScene when scene is system', async () => {
    mockUseStore.mockImplementation((selector?: (s: unknown) => unknown) => {
      if (typeof selector === 'function') return selector({ currentScene: 'system' })
      return { currentScene: 'system' }
    })
    render(wrapWithSuspense(<SceneSelector />))
    expect(await screen.findByTestId('system-scene')).toBeDefined()
  })

  it('renders TerminalView for terminal scene without dashboard', () => {
    mockUseStore.mockImplementation((selector?: (s: unknown) => unknown) => {
      if (typeof selector === 'function') return selector({ currentScene: 'terminal', slideOverContent: null })
      return { currentScene: 'terminal', slideOverContent: null }
    })
    render(wrapWithSuspense(<SceneSelector />))
    expect(screen.getByTestId('terminal-view')).toBeDefined()
  })

  it('renders DashboardView for terminal scene with dashboard slideOver', async () => {
    mockUseStore.mockImplementation((selector?: (s: unknown) => unknown) => {
      if (typeof selector === 'function') return selector({ currentScene: 'terminal', slideOverContent: { type: 'dashboard' } })
      return { currentScene: 'terminal', slideOverContent: { type: 'dashboard' } }
    })
    render(wrapWithSuspense(<SceneSelector />))
    expect(await screen.findByTestId('dashboard-view')).toBeDefined()
  })
})
