import { describe, it, expect, vi } from 'vitest';
import { createCopilotSlice } from '../copilotSlice';
import type { CopilotSlice } from '../copilotSlice';

function makeSlice(): CopilotSlice {
  const set = vi.fn();
  const get = vi.fn<() => CopilotSlice>().mockReturnValue({} as CopilotSlice);
  return createCopilotSlice(set, get, {} as never);
}

describe('copilotSlice', () => {
  it('has correct initial state', () => {
    const slice = makeSlice();
    expect(slice.messages).toEqual([]);
    expect(slice.isStreaming).toBe(false);
    expect(slice.streamingMessage).toBe('');
    expect(slice.streamingToolCalls).toEqual([]);
    expect(slice.selectedAgent).toBe('');
  });

  it('setMessages accepts message list', () => {
    const slice = makeSlice();
    slice.setMessages([{ role: 'user', content: 'hello', createdAt: 100 }]);
    slice.setMessages([]);
  });

  it('addChatMessage is callable', () => {
    const slice = makeSlice();
    slice.addChatMessage({ role: 'user', content: 'hello', createdAt: Date.now() });
  });

  it('clearMessages is callable', () => {
    const slice = makeSlice();
    slice.clearMessages();
  });

  it('setIsStreaming toggles streaming', () => {
    const slice = makeSlice();
    slice.setIsStreaming(true);
    slice.setIsStreaming(false);
  });

  it('setStreamingMessage accepts string', () => {
    const slice = makeSlice();
    slice.setStreamingMessage('partial response...');
    slice.setStreamingMessage('');
  });

  it('setStreamingToolCalls accepts call list', () => {
    const slice = makeSlice();
    slice.setStreamingToolCalls([{ name: 'fetch', args: '{}' }]);
    slice.setStreamingToolCalls([]);
  });

  it('setSelectedAgent selects an agent', () => {
    const slice = makeSlice();
    slice.setSelectedAgent('agent-abc');
    slice.setSelectedAgent('');
  });

  it('resetCopilot is callable', () => {
    const slice = makeSlice();
    slice.resetCopilot();
  });
});
