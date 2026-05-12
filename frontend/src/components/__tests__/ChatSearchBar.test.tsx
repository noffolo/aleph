import React from 'react'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'copilot.search': 'Cerca nella chat...',
    }
    return map[key] ?? key
  },
}))

import { ChatSearchBar } from '../ChatSearchBar'

describe('ChatSearchBar', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders search input with placeholder', () => {
    render(
      <ChatSearchBar query="" setQuery={vi.fn()} matchCount={0} />
    )
    expect(screen.getByPlaceholderText('Cerca nella chat...')).toBeInTheDocument()
  })

  it('renders with initial query value', () => {
    render(
      <ChatSearchBar query="hello" setQuery={vi.fn()} matchCount={0} />
    )
    const input = screen.getByPlaceholderText('Cerca nella chat...') as HTMLInputElement
    expect(input.value).toBe('hello')
  })

  it('calls setQuery after debounce delay (300ms)', () => {
    const setQuery = vi.fn()
    render(
      <ChatSearchBar query="" setQuery={setQuery} matchCount={0} />
    )
    const input = screen.getByPlaceholderText('Cerca nella chat...')
    fireEvent.change(input, { target: { value: 'test' } })

    expect(setQuery).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(300)
    })
    expect(setQuery).toHaveBeenCalledWith('test')
  })

  it('shows clear button when query is non-empty', () => {
    render(
      <ChatSearchBar query="text" setQuery={vi.fn()} matchCount={0} />
    )
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBeGreaterThan(0)
  })

  it('clears query on clear button click', () => {
    const setQuery = vi.fn()
    render(
      <ChatSearchBar query="text" setQuery={setQuery} matchCount={0} />
    )
    const clearBtn = screen.getByRole('button')
    fireEvent.click(clearBtn)
    expect(setQuery).toHaveBeenCalledWith('')
  })

  it('does not show clear button when query is empty', () => {
    const { container } = render(
      <ChatSearchBar query="" setQuery={vi.fn()} matchCount={0} />
    )
    expect(container.querySelector('button')).not.toBeInTheDocument()
  })

  it('shows match count when greater than 0', () => {
    render(
      <ChatSearchBar query="test" setQuery={vi.fn()} matchCount={5} />
    )
    expect(screen.getByText('5 risultati')).toBeInTheDocument()
  })

  it('hides match count when equal to 0', () => {
    render(
      <ChatSearchBar query="test" setQuery={vi.fn()} matchCount={0} />
    )
    expect(screen.queryByText(/risultati/)).not.toBeInTheDocument()
  })

  it('renders search icon', () => {
    const { container } = render(
      <ChatSearchBar query="" setQuery={vi.fn()} matchCount={0} />
    )
    expect(container.querySelector('svg')).toBeInTheDocument()
  })
})
