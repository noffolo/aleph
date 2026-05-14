import { describe, it, expect } from 'vitest';
import { useStore } from '../useStore';

describe('Store Integration', () => {
  it('should create the store and initialize state', () => {
    const state = useStore.getState();
    
    expect(state).toBeDefined();
    expect(state.projectID).toBe('');
    expect(state.currentView).toBe('copilot');
    expect(state.messages).toEqual([]);
    expect(state.sandboxInput).toBe('{}');
    expect(state.ollamaHealthy).toBe(false);
    expect(state.showOnboarding).toBe(true);
  });

  it('should allow updates to the store', () => {
    useStore.setState({ projectID: 'test-proj' });
    expect(useStore.getState().projectID).toBe('test-proj');
  });

  it('setProjectContext resets all slices and sets projectID', () => {
    const store = useStore;

    store.getState().setProjectContext('new-project');

    const state = store.getState();
    expect(state.projectID).toBe('new-project');
    expect(state.apiKeys).toEqual([]);
    expect(state.messages).toEqual([]);
    expect(state.isStreaming).toBe(false);
    expect(state.ollamaHealthy).toBe(false);
    expect(state.toastMessages).toEqual([]);
    expect(state.currentScene).toBeNull();
  });

  it('setProjectContext calls reset functions which restore defaults', () => {
    const store = useStore;
    
    // setProjectContext calls resetUI() which resets enableScanline to true
    store.getState().setProjectContext('another-project');

    const state = store.getState();
    expect(state.projectID).toBe('another-project');
    expect(state.showOnboarding).toBe(true);
    expect(state.enableScanline).toBe(true);
    expect(state.apiKeys).toEqual([]);
    expect(state.messages).toEqual([]);
  });

  it('uiSlice has lastError and setLastError', () => {
    const state = useStore.getState();

    expect(state).toHaveProperty('lastError');
    expect(state).toHaveProperty('setLastError');
    expect(state.lastError).toBeNull();

    state.setLastError('test error');
    expect(useStore.getState().lastError).toBe('test error');

    state.setLastError(null);
    expect(useStore.getState().lastError).toBeNull();
  });
});
