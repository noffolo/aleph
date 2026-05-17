import { describe, it, expect, vi } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'

function useSimpleHook({ onLoad }: { onLoad: () => Promise<{ items: string[] }> }) {
  const [items, setItems] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  
  const loadMore = useCallback(async () => {
    if (loading) return
    setLoading(true)
    try {
      const result = await onLoad()
      setItems(result.items)
    } finally {
      setLoading(false)
    }
  }, [loading, onLoad])
  
  return { items, loading, loadMore }
}

import { useState, useCallback } from 'react'

describe('Minimal async hook test', () => {
  it('loads items', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ items: ['a', 'b'] })
    const { result } = renderHook(() => useSimpleHook({ onLoad: mockFetch }))
    
    await act(async () => {
      result.current.loadMore()
    })
    
    expect(result.current.items).toEqual(['a', 'b'])
  })
  
  it('does not load when loading', async () => {
    let resolve!: (v: any) => void
    const deferred = new Promise(resolve_ => { resolve = resolve_ })
    const mockFetch = vi.fn().mockImplementation(() => deferred)
    
    const { result } = renderHook(() => useSimpleHook({ onLoad: mockFetch }))
    
    await act(async () => {
      result.current.loadMore()
    })
    // Should be true since setLoading happened before await
    expect(result.current.loading).toBe(true)
  })
})
