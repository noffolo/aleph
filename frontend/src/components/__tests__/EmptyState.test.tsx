import React from 'react'
import { render, screen, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { EmptyState } from '../ui/EmptyState'

describe('EmptyState', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders first ghost prompt after mount effects fire', () => {
    render(<EmptyState />)
    expect(screen.getByText(/Prova \/explore per esplorare dati/)).toBeInTheDocument()
  })

  it('has transition and animation classes on the container', () => {
    render(<EmptyState />)
    const div = screen.getByText(/aleph-v2/)
    expect(div.className).toContain('animate-fade-in')
    expect(div.className).toContain('transition-opacity')
  })

  it('cycles ghost prompts at 4000ms intervals', () => {
    render(<EmptyState />)

    act(() => { vi.advanceTimersByTime(4000) })
    const divAfter4000 = screen.getByText(/aleph-v2/)
    expect(divAfter4000.className).toContain('opacity-0')

    act(() => { vi.advanceTimersByTime(500) })
    expect(screen.getByText(/Usa \/agent per parlare con un agente/)).toBeInTheDocument()
  })

  it('clears interval on unmount', () => {
    const { unmount } = render(<EmptyState />)
    unmount()
    act(() => { vi.advanceTimersByTime(10000) })
  })

  it('cycles through all six ghost prompts', () => {
    render(<EmptyState />)

    const patterns = [
      /Prova \/explore/,
      /Usa \/agent/,
      /\/help mostra/,
      /Cerca \/tools/,
      /\/predict/,
      /\/library/,
    ]

    for (let i = 1; i < patterns.length; i++) {
      act(() => { vi.advanceTimersByTime(4500) })
      expect(screen.getByText(patterns[i])).toBeInTheDocument()
    }
  })

  it('wraps around to first prompt after full cycle', () => {
    render(<EmptyState />)
    for (let i = 0; i < 6; i++) {
      act(() => { vi.advanceTimersByTime(4500) })
    }
    expect(screen.getByText(/Prova \/explore/)).toBeInTheDocument()
  })
})
