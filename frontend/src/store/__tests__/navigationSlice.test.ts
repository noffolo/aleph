import { describe, it, expect, vi } from 'vitest';
import { createNavigationSlice } from '../navigationSlice';

const createMockSet = () => {
  let state: any = { commandHistory: [] };
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

describe('navigationSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    expect(slice.currentView).toBe('copilot');
    expect(slice.inlineContent).toBeNull();
    expect(slice.showInlinePanel).toBe(false);
    expect(slice.slideOverContent).toBeNull();
    expect(slice.isCommandPaletteOpen).toBe(false);
    expect(slice.activeView).toBe('table');
  });

  it('should update views and content', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    slice.setCurrentView('inline');
    expect(set).toHaveBeenCalled();
    slice.setActiveView('graph');
    expect(set).toHaveBeenCalledTimes(2);
    slice.setShowInlinePanel(true);
    expect(set).toHaveBeenCalledTimes(3);
  });

  it('should add to command history', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    slice.addToHistory('test command');
    expect(set).toHaveBeenCalled();
  });

  it('should reset navigation', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    slice.resetNavigation();
    expect(set).toHaveBeenCalled();
  });
});
