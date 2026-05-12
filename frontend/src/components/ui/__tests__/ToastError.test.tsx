import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

const { tMock } = vi.hoisted(() => {
  const fn = vi.fn((key: string) => {
    const map: Record<string, string> = {
      'inlineError.dismiss': 'Dismiss',
      'toast.retry': 'Retry',
    }
    return map[key] ?? key
  })
  return { tMock: fn }
})

vi.mock('../../i18n', () => ({
  t: tMock,
}))

import { ToastError } from '../ToastError'

describe('ToastError', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    tMock.mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the error message', () => {
    const onDismiss = vi.fn()
    render(<ToastError message="Connection failed" onDismiss={onDismiss} />)
    expect(screen.getByText('Connection failed')).toBeInTheDocument()
  })

  it('calls onDismiss after 5 seconds', () => {
    const onDismiss = vi.fn()
    render(<ToastError message="error" onDismiss={onDismiss} />)
    vi.advanceTimersByTime(5000)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('renders dismiss button and calls onDismiss on click', () => {
    const onDismiss = vi.fn()
    render(<ToastError message="error" onDismiss={onDismiss} />)
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[0])
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('does not render retry button when onRetry is not provided', () => {
    const onDismiss = vi.fn()
    render(<ToastError message="error" onDismiss={onDismiss} />)
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBe(1)
  })

  it('renders retry button when onRetry is provided', () => {
    const onDismiss = vi.fn()
    const onRetry = vi.fn()
    render(<ToastError message="error" onDismiss={onDismiss} onRetry={onRetry} />)
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBe(2)
  })

  it('calls onRetry when retry button is clicked', () => {
    const onDismiss = vi.fn()
    const onRetry = vi.fn()
    render(<ToastError message="error" onDismiss={onDismiss} onRetry={onRetry} />)
    const buttons = screen.getAllByRole('button')
    const retryBtn = buttons[0]  
    fireEvent.click(retryBtn)
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('has aria-live assertive for accessibility', () => {
    const onDismiss = vi.fn()
    const { container } = render(<ToastError message="error" onDismiss={onDismiss} />)
    const live = container.querySelector('[aria-live="assertive"]')
    expect(live).toBeInTheDocument()
  })

  it('has the alert role on the inner container', () => {
    const onDismiss = vi.fn()
    render(<ToastError message="error" onDismiss={onDismiss} />)
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })
})
