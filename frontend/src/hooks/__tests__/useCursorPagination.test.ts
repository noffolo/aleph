import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useCursorPagination } from '../useCursorPagination'

vi.mock('../../lib/errorReporter', () => ({
  reportError: vi.fn(),
}))

describe('useCursorPagination', () => {
  const mockClientMethod = vi.fn()
  const mockRequestBuilder = vi.fn()
  const mockResponseExtractor = vi.fn()
  const mockStoreSetter = vi.fn()

  const defaultProps = () => ({
    clientMethod: mockClientMethod,
    requestBuilder: mockRequestBuilder,
    responseExtractor: mockResponseExtractor,
    storeSetter: mockStoreSetter,
    initialItems: [{ id: '1', name: 'Item 1' }],
  })

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns initial items', () => {
    const { result } = renderHook(() => useCursorPagination(defaultProps()))
    expect(result.current.items).toEqual([{ id: '1', name: 'Item 1' }])
    expect(result.current.hasMore).toBe(true)
    expect(result.current.loading).toBe(false)
  })

  it('loads more items via loadMore', async () => {
    const request = { cursor: '', limit: 10 }
    const response = { items: [], nextCursor: '' }
    mockRequestBuilder.mockReturnValue(request)
    mockClientMethod.mockResolvedValue(response)
    mockResponseExtractor.mockReturnValue({ items: [{ id: '2', name: 'Item 2' }], nextCursor: '' })

    const { result } = renderHook(() => useCursorPagination(defaultProps()))

    act(() => {
      result.current.loadMore()
    })

    await waitFor(() => {
      expect(result.current.items).toHaveLength(2)
    })

    expect(result.current.hasMore).toBe(false)
    expect(mockStoreSetter).toHaveBeenCalledWith([{ id: '1', name: 'Item 1' }, { id: '2', name: 'Item 2' }])
  })

  it('sets hasMore true when nextCursor is provided', async () => {
    mockRequestBuilder.mockReturnValue({ cursor: '' })
    mockClientMethod.mockResolvedValue({})
    mockResponseExtractor.mockReturnValue({ items: [{ id: '3', name: 'Item 3' }], nextCursor: 'abc' })

    const { result } = renderHook(() => useCursorPagination(defaultProps()))

    act(() => {
      result.current.loadMore()
    })

    await waitFor(() => {
      expect(result.current.hasMore).toBe(true)
    })
  })

  it('does not load more when already loading', async () => {
    mockRequestBuilder.mockReturnValue({ cursor: '' })
    mockClientMethod.mockImplementation(() => new Promise(() => {})) // never resolves

    const { result } = renderHook(() => useCursorPagination(defaultProps()))

    act(() => {
      result.current.loadMore()
    })

    expect(result.current.loading).toBe(true)

    act(() => {
      result.current.loadMore() // should be no-op
    })

    expect(mockClientMethod).toHaveBeenCalledTimes(1)
  })

  it('does not load more when hasMore is false', async () => {
    mockRequestBuilder.mockReturnValue({ cursor: '' })
    mockClientMethod.mockResolvedValue({})
    mockResponseExtractor.mockReturnValue({ items: [], nextCursor: '' })

    const { result } = renderHook(() => useCursorPagination(defaultProps()))

    act(() => {
      result.current.loadMore()
    })
    await waitFor(() => { expect(result.current.hasMore).toBe(false) })

    act(() => {
      result.current.loadMore()
    })

    expect(mockClientMethod).toHaveBeenCalledTimes(1)
  })

  it('reports error on failure', async () => {
    const { reportError } = await import('../../lib/errorReporter')
    mockRequestBuilder.mockReturnValue({ cursor: '' })
    mockClientMethod.mockRejectedValue(new Error('Network error'))

    const { result } = renderHook(() => useCursorPagination(defaultProps()))

    act(() => {
      result.current.loadMore()
    })

    await waitFor(() => {
      expect(reportError).toHaveBeenCalledWith('useCursorPagination', expect.any(Error))
    })
    expect(result.current.loading).toBe(false)
  })

  it('updates items when initialItems changes', () => {
    const { result, rerender } = renderHook(
      ({ initialItems }) => useCursorPagination({ ...defaultProps(), initialItems }),
      { initialProps: { initialItems: [{ id: '1', name: 'Item 1' }] } }
    )

    expect(result.current.items).toEqual([{ id: '1', name: 'Item 1' }])

    rerender({ initialItems: [{ id: 'X', name: 'Updated' }] })

    expect(result.current.items).toEqual([{ id: 'X', name: 'Updated' }])
  })
})
