import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TerminalProgressBar } from '../terminal/TerminalProgressBar'

vi.mock('lucide-react', () => ({
  Check: () => null,
  XCircle: () => null,
  AlertTriangle: () => null,
  Pause: () => null,
  Loader2: () => null,
  Clock: () => null,
  Zap: () => null,
}))

describe('TerminalProgressBar', () => {
  it('renders classic variant with label and percentage', () => {
    render(<TerminalProgressBar label="Processing" percent={45} variant="classic" />)
    expect(screen.getByText(/Processing/)).toBeInTheDocument()
    expect(screen.getByText(/45%/)).toBeInTheDocument()
  })

  it('renders compact variant', () => {
    const { container } = render(<TerminalProgressBar label="Compact" percent={30} variant="compact" />)
    expect(container.textContent).toContain('30%')
  })

  it('renders nested variant', () => {
    const { container } = render(<TerminalProgressBar label="Nested" percent={60} variant="nested" />)
    expect(container.textContent).toContain('60%')
  })

  it('renders full variant', () => {
    const { container } = render(<TerminalProgressBar label="Full" percent={75} variant="full" />)
    expect(container.textContent).toContain('75%')
  })

  it('renders running status with spinner', () => {
    render(<TerminalProgressBar label="Running" percent={0} status="running" />)
    expect(screen.getByText(/Running/)).toBeInTheDocument()
  })

  it('renders done status', () => {
    render(<TerminalProgressBar label="Done" percent={100} status="done" />)
    expect(screen.getByText(/Done/)).toBeInTheDocument()
  })

  it('renders error status', () => {
    render(<TerminalProgressBar label="Error" percent={0} status="error" />)
    expect(screen.getByText(/Error/)).toBeInTheDocument()
  })

  it('renders paused status', () => {
    render(<TerminalProgressBar label="Paused" percent={50} status="paused" />)
    expect(screen.getByText(/Paused/)).toBeInTheDocument()
  })
})
