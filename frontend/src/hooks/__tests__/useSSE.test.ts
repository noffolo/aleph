import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSSE } from '../useSSE';
import { useStore } from '@/store/useStore';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(() => ({})), { subscribe: vi.fn(() => vi.fn()) }),
}));

describe('useSSE', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (useStore as any).mockReturnValue({});
    global.EventSource = vi.fn() as any;
  });

  it('returns connect and disconnect functions', () => {
    const { result } = renderHook(() => useSSE());
    expect(typeof result.current.connect).toBe('function');
    expect(typeof result.current.disconnect).toBe('function');
  });
});
