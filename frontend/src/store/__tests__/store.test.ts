import { describe, it, expect } from 'vitest';
import { useStore } from '../useStore';

describe('Store Integration', () => {
  it('should create the store and initialize state', () => {
    const state = useStore.getState();
    
    expect(state).toBeDefined();
    expect(state.projectID).toBe('');
    expect(state.currentView).toBe('copilot');
    expect(state.chat).toEqual([]);
    expect(state.sandboxInput).toBe('{}');
    expect(state.ollamaHealthy).toBe(false);
    expect(state.showOnboarding).toBe(true);
  });

  it('should allow updates to the store', () => {
    useStore.setState({ projectID: 'test-proj' });
    expect(useStore.getState().projectID).toBe('test-proj');
  });
});
