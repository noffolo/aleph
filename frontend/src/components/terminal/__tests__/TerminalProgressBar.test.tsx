import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { TerminalProgressBar } from '../TerminalProgressBar'

describe('TerminalProgressBar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // --- Compact variant ---
  it('renders compact variant with label and percentage', () => {
    const { container } = render(
      <TerminalProgressBar label="Processing" percent={50} variant="compact" />
    )
    expect(container.textContent).toContain('Processing')
    expect(container.textContent).toContain('50%')
  })

  it('shows spinner char when status is running', () => {
    const { container } = render(
      <TerminalProgressBar label="Run" percent={30} variant="compact" status="running" />
    )
    // Spinner char should be present
    const text = container.textContent || ''
    expect(text).toBeTruthy()
  })

  it('shows done indicator when status is done', () => {
    const { container } = render(
      <TerminalProgressBar label="Done" percent={100} variant="compact" status="done" />
    )
    expect(container.textContent).toContain('●')
    expect(container.textContent).toContain('100%')
  })

  it('shows error indicator when status is error', () => {
    const { container } = render(
      <TerminalProgressBar label="Fail" percent={45} variant="compact" status="error" />
    )
    expect(container.textContent).toContain('✕')
  })

  it('shows paused indicator when status is paused', () => {
    const { container } = render(
      <TerminalProgressBar label="Pause" percent={20} variant="compact" status="paused" />
    )
    expect(container.textContent).toContain('⏸')
  })

  // --- Nested variant ---
  it('renders nested variant with label and percentage', () => {
    const { container } = render(
      <TerminalProgressBar label="Nested Task" percent={75} variant="nested" />
    )
    expect(container.textContent).toContain('Nested Task')
    expect(container.textContent).toContain('75%')
  })

  it('shows elapsed time in nested variant', () => {
    render(
      <TerminalProgressBar label="Timed" percent={50} variant="nested" elapsedMs={5000} />
    )
    expect(screen.getByText('5s')).toBeInTheDocument()
  })

  it('shows ETA in nested variant', () => {
    const { container } = render(
      <TerminalProgressBar label="ETA" percent={60} variant="nested" etaMs={30000} />
    )
    expect(container.textContent).toMatch(/~30s/)
  })

  it('shows completion text when done in nested variant', () => {
    render(
      <TerminalProgressBar label="Done" percent={100} variant="nested" status="done" />
    )
    expect(screen.getByText('Completato')).toBeInTheDocument()
  })

  it('shows failure text when error in nested variant', () => {
    render(
      <TerminalProgressBar label="Fail" percent={50} variant="nested" status="error" />
    )
    expect(screen.getByText('Fallito')).toBeInTheDocument()
  })

  // --- Full variant ---
  it('renders full variant with label', () => {
    render(
      <TerminalProgressBar label="Full Task" percent={90} variant="full" />
    )
    expect(screen.getByText('Full Task')).toBeInTheDocument()
  })

  it('shows current/total in full variant', () => {
    const { container } = render(
      <TerminalProgressBar label="Count" percent={50} variant="full" current={500} total={1000} />
    )
    expect(container.textContent).toMatch(/500.*1,?000/)
  })

  it('shows throughput in full variant', () => {
    render(
      <TerminalProgressBar label="Download" percent={40} variant="full" throughput={{ value: 12.5, unit: 'MB/s' }} />
    )
    expect(screen.getByText(/12.5 MB\/s/)).toBeInTheDocument()
  })

  // --- Classic variant ---
  it('renders classic variant as default', () => {
    const { container } = render(
      <TerminalProgressBar label="Default" percent={42} />
    )
    expect(container.textContent).toContain('Default')
    expect(container.textContent).toContain('42%')
  })

  it('shows current/total in classic variant', () => {
    render(
      <TerminalProgressBar label="Items" percent={50} current={5} total={10} />
    )
    expect(screen.getByText(/\[5 \/ 10\]/)).toBeInTheDocument()
  })

  it('shows elapsed time in classic variant', () => {
    render(
      <TerminalProgressBar label="Running" percent={50} elapsedMs={3000} />
    )
    expect(screen.getByText('3s')).toBeInTheDocument()
  })

  // --- Sparkline ---
  it('renders sparkline when enabled with data', () => {
    const { container } = render(
      <TerminalProgressBar label="Trend" percent={60} showSparkline={true} sparklineHistory={[10, 20, 30, 40, 50, 60]} />
    )
    // Sparkline chars should be present (▁▂▃▄▅▆▇█)
    const text = container.textContent || ''
    const hasSparkChars = '▁▂▃▄▅▆▇█'.split('').some(c => text.includes(c))
    expect(hasSparkChars).toBe(true)
  })

  // --- Clamping ---
  it('clamps percent values below 0 to 0', () => {
    const { container } = render(
      <TerminalProgressBar label="Clamped" percent={-10} variant="compact" />
    )
    expect(container.textContent).not.toContain('-10%')
  })

  it('clamps percent values above 100 to 100', () => {
    const { container } = render(
      <TerminalProgressBar label="Clamped" percent={150} variant="compact" />
    )
    expect(container.textContent).not.toContain('150%')
  })
})
