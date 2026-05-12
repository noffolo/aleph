import { describe, it, expect, vi } from 'vitest';
import { createCopilotSlice } from '../copilotSlice';

const createMockSet = () => {
  let state: any = { messages: [] };
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
    
    expect(slice.messages).toEqual([]);
    expect(slice.isStreaming).toBe(false);
    expect(slice.streamingMessage).toBe('');
    expect(slice.streamingToolCalls).toEqual([]);
    expect(slice.selectedAgent).toBe('');
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
    const result = updateFn({ messages: [] });
    expect(result.messages[0]).toMatchObject(msg);
  });

  it('should clear messages', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    slice.clearMessages();
    expect(set).toHaveBeenCalled();
  });

  it('should reset copilot state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createCopilotSlice(set, get, {} as any);
    
    slice.resetCopilot();
    expect(set).toHaveBeenCalled();
  });
});
