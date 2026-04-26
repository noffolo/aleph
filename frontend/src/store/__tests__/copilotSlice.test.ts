import { describe, it, expect, vi } from 'vitest';
import { createCopilotSlice } from '../copilotSlice';

const createMockSet = () => {
  let state: any = { chat: [], bookmarkedIds: new Set() };
  const mockSet = vi.fn((update: any) => {
    if (typeof update === 'function') {
      const result = update(state);
      state = { ...state, ...result };
      return state;
    }
    state = { ...state, ...update };
    return state;
  });
  return mockSet;
};

describe('copilotSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    
    expect(slice.chat).toEqual([]);
    expect(slice.input).toBe('');
    expect(slice.isStreaming).toBe(false);
    expect(slice.streamAbortController).toBeNull();
    expect(slice.pendingConfirmation).toBeNull();
    expect(slice.selectedAgent).toBe('');
    expect(slice.splitView).toBe(false);
    expect(slice.bookmarkedIds).toBeInstanceOf(Set);
    expect(slice.chatSearchQuery).toBe('');
    expect(slice.onlyBookmarks).toBe(false);
  });

  it('should add chat message', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    const msg = { role: 'user' as const, content: 'hello', createdAt: Date.now() };
    
    slice.addChatMessage(msg);
    expect(set).toHaveBeenCalled();
    const updateFn = set.mock.calls[0][0];
    expect(typeof updateFn).toBe('function');
    const result = updateFn({ chat: [] });
    expect(result.chat[0]).toMatchObject(msg);
  });

  it('should clear chat', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    slice.clearChat();
    expect(set).toHaveBeenCalled();
  });

  it('should toggle bookmarks', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    
    slice.toggleBookmark(1);
    expect(set).toHaveBeenCalledTimes(1);
    slice.toggleBookmark(1);
    expect(set).toHaveBeenCalledTimes(2);
  });

  it('should cancel stream', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    const controller = new AbortController();
    const spy = vi.spyOn(controller, 'abort');
    
    slice.setStreamAbortController(controller);
    slice.cancelStream();
    
    expect(spy).toHaveBeenCalled();
  });

  it('should reset copilot state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    
    slice.resetCopilot();
    expect(set).toHaveBeenCalled();
  });
});
