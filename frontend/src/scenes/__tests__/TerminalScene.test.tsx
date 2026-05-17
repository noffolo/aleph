import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TerminalScene } from '../TerminalScene'

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['', vi.fn()]),
}))

vi.mock('../../components/terminal/TerminalView', () => ({
  TerminalView: () => <div data-testid="terminal-view">Terminal</div>,
}))

vi.mock('../../components/DashboardView', () => ({
  DashboardView: () => <div data-testid="dashboard-view">Dashboard</div>,
}))

vi.mock('../../components/SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading</div>,
}))

describe('TerminalScene', () => {
  it('renders TerminalView by default', () => {
    render(<TerminalScene />)
    expect(screen.getByTestId('terminal-view')).toBeDefined()
  })
})
