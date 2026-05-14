import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useDebounce } from '../useDebounce'

describe('useDebounce', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns initial value immediately', () => {
    const { result } = renderHook(() => useDebounce('hello', 300))
    expect(result.current).toBe('hello')
  })

  it('returns initial value for other types', () => {
    const { result } = renderHook(() => useDebounce(42, 500))
    expect(result.current).toBe(42)
  })

  it('does not update before delay', () => {
    const { result, rerender } = renderHook(({ value, delay }: { value: string, delay: number }) => useDebounce(value, delay), {
      initialProps: { value: 'hello', delay: 500 },
    })
    rerender({ value: 'world', delay: 500 })
    act(() => { vi.advanceTimersByTime(400) })
    expect(result.current).toBe('hello')
  })

  it('updates after delay', () => {
    const { result, rerender } = renderHook(({ value, delay }: { value: string, delay: number }) => useDebounce(value, delay), {
      initialProps: { value: 'hello', delay: 500 },
    })
    rerender({ value: 'world', delay: 500 })
    act(() => { vi.advanceTimersByTime(500) })
    expect(result.current).toBe('world')
  })

  it('resets timer on new value', () => {
    const { result, rerender } = renderHook(({ value, delay }: { value: string, delay: number }) => useDebounce(value, delay), {
      initialProps: { value: 'a', delay: 500 },
    })
    rerender({ value: 'b', delay: 500 })
    act(() => { vi.advanceTimersByTime(300) })
    rerender({ value: 'c', delay: 500 })
    act(() => { vi.advanceTimersByTime(300) })
    expect(result.current).toBe('a')
    act(() => { vi.advanceTimersByTime(200) })
    expect(result.current).toBe('c')
  })

  it('changes delay on rerender', () => {
    const { result, rerender } = renderHook(({ value, delay }: { value: string, delay: number }) => useDebounce(value, delay), {
      initialProps: { value: 'x', delay: 1000 },
    })
    rerender({ value: 'y', delay: 100 })
    act(() => { vi.advanceTimersByTime(100) })
    expect(result.current).toBe('y')
  })
})
