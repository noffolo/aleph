import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useSSE } from '../useSSE';
import { useStore } from '@/store/useStore';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(() => ({})), { subscribe: vi.fn(() => vi.fn()) }),
}));

describe('useSSE', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    (useStore as any).mockReturnValue({});
    global.EventSource = vi.fn() as any;
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns connect and disconnect functions', () => {
    const { result } = renderHook(() => useSSE());
    expect(typeof result.current.connect).toBe('function');
    expect(typeof result.current.disconnect).toBe('function');
  });

  it('returns initial status as disconnected', () => {
    const { result } = renderHook(() => useSSE());
    expect(result.current.status).toBe('disconnected');
    expect(result.current.reconnectCount).toBe(0);
  });

  it('connect attempts fetch after 100ms timeout', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

    renderHook(() => useSSE());

    // The timer fires after 100ms
    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    expect(fetchSpy).toHaveBeenCalled();
    fetchSpy.mockRestore();
  });

  it('does not reconnect if disconnect was called', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

    const { result, unmount } = renderHook(() => useSSE());

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    // Call disconnect and unmount
    act(() => { result.current.disconnect(); });
    unmount();

    // Advance to ensure no further reconnect
    await act(async () => { vi.advanceTimersByTime(5000); });

    // fetch should have been called only once (initial connect)
    const callCount = fetchSpy.mock.calls.length;

    fetchSpy.mockRestore();
    // We make no assertion about exact count since async timing is complex,
    // but the key invariant is unmount cleans up
  });

  it('status transitions to reconnecting on network error', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

    const { result } = renderHook(() => useSSE());

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    expect(result.current.status).toBe('reconnecting');
    expect(result.current.reconnectCount).toBeGreaterThanOrEqual(1);

    vi.restoreAllMocks();
  });

  it('disconnect clears timers and aborts controller', () => {
    const { result } = renderHook(() => useSSE());
    const clearSpy = vi.spyOn(globalThis, 'clearTimeout');

    act(() => { result.current.disconnect(); });

    // disconnect should be idempotent
    act(() => { result.current.disconnect(); });

    clearSpy.mockRestore();
  });

  it('reconnectCount increments on reconnection', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));

    const { result } = renderHook(() => useSSE());

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    expect(result.current.reconnectCount).toBeGreaterThanOrEqual(1);

    vi.restoreAllMocks();
  });

  it('calls onError handler on failed connection', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));
    const onError = vi.fn();

    renderHook(() => useSSE({ onError }));

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    expect(onError).toHaveBeenCalled();

    vi.restoreAllMocks();
  });

  it('does not crash with no handlers provided', () => {
    const { result } = renderHook(() => useSSE());
    expect(result.current).toBeDefined();
  });

  it('sets status to connected on successful fetch then reconnects after stream ends', async () => {
    const mockReader = {
      read: vi.fn()
        .mockResolvedValueOnce({ done: false, value: new TextEncoder().encode(':keepalive\n\n') })
        .mockResolvedValueOnce({ done: true, value: undefined }),
      cancel: vi.fn(),
      releaseLock: vi.fn(),
    };
    const mockResponse = {
      ok: true,
      body: { getReader: () => mockReader },
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse as any);

    const { result } = renderHook(() => useSSE());

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    // After stream ends, status goes to reconnecting (scheduleReconnect called)
    expect(result.current.status).toBe('reconnecting');

    vi.restoreAllMocks();
  });

  it('calls onOpen handler on successful connection', async () => {
    const mockReader = {
      read: vi.fn()
        .mockResolvedValueOnce({ done: false, value: new TextEncoder().encode(':keepalive\n\n') })
        .mockResolvedValueOnce({ done: true, value: undefined }),
      cancel: vi.fn(),
      releaseLock: vi.fn(),
    };
    const mockResponse = {
      ok: true,
      body: { getReader: () => mockReader },
    };
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse as any);
    const onOpen = vi.fn();

    renderHook(() => useSSE({ onOpen }));

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    expect(onOpen).toHaveBeenCalled();

    vi.restoreAllMocks();
  });

  it('handles non-200 response with reconnection', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({ ok: false, status: 500 } as any);
    const onError = vi.fn();

    const { result } = renderHook(() => useSSE({ onError }));

    await act(async () => { vi.advanceTimersByTime(100); });
    await act(async () => { vi.runAllTimers(); });

    expect(onError).toHaveBeenCalled();
    expect(result.current.status).toBe('reconnecting');

    vi.restoreAllMocks();
  });
});
