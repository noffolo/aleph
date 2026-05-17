import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useCursorPagination } from '../useCursorPagination'

vi.mock('../../lib/errorReporter', () => ({ reportError: vi.fn() }))

describe('useCursorPagination', () => {
  const clientMethod = vi.fn()
  const requestBuilder = vi.fn()
  const responseExtractor = vi.fn()
  const storeSetter = vi.fn()

  const defaultProps = {
    clientMethod,
    requestBuilder,
    responseExtractor,
    storeSetter,
  }

  beforeEach(() => { vi.clearAllMocks() })

  it('returns initial items with default state', () => {
    const { result } = renderHook(() =>
      useCursorPagination({ ...defaultProps, initialItems: [{ id: '1', name: 'Item 1' }] })
    )
    expect(result.current.items).toEqual([{ id: '1', name: 'Item 1' }])
    expect(result.current.hasMore).toBe(true)
    expect(result.current.loading).toBe(false)
  })

  it('updates items when initialItems changes', () => {
    const { result, rerender } = renderHook(
      ({ initialItems }) => useCursorPagination({ ...defaultProps, initialItems }),
      { initialProps: { initialItems: [{ id: '1' }] } }
    )
    expect(result.current.items).toEqual([{ id: '1' }])

    rerender({ initialItems: [{ id: 'X' }] })
    expect(result.current.items).toEqual([{ id: 'X' }])
  })

  it('synchronous getters: loadMore is a function', () => {
    const { result } = renderHook(() =>
      useCursorPagination({ ...defaultProps, initialItems: [] })
    )
    expect(typeof result.current.loadMore).toBe('function')
    expect(result.current.hasMore).toBe(true)
    expect(result.current.loading).toBe(false)
  })
})
