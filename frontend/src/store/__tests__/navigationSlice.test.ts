import { describe, it, expect, vi } from 'vitest';
import { createNavigationSlice } from '../navigationSlice';

const createMockSet = () => {
  let state: any = {};
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
    expect(slice.slideOverContent).toBeNull();
    expect(slice.isCommandPaletteOpen).toBe(false);
  });

  it('should update currentScene and currentView', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    slice.setCurrentView('copilot');
    expect(set).toHaveBeenCalled();
    slice.setCurrentScene('explore');
    expect(set).toHaveBeenCalledTimes(2);
  });

  it('should set and clear slideOverContent', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    slice.setSlideOverContent({ type: 'explore', title: 'Test' });
    expect(set).toHaveBeenCalled();
  });

  it('should reset navigation', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);
    
    slice.resetNavigation();
    expect(set).toHaveBeenCalled();
  });

  it('should set inline content', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);

    slice.setInlineContent({ type: 'explore', payload: { id: '1' } });
    expect(set).toHaveBeenCalledWith({ inlineContent: { type: 'explore', payload: { id: '1' } } });
  });

  it('should toggle inline panel visibility', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);

    slice.setShowInlinePanel(true);
    expect(set).toHaveBeenCalledWith({ showInlinePanel: true });
    slice.setShowInlinePanel(false);
    expect(set).toHaveBeenCalledWith({ showInlinePanel: false });
  });

  it('should toggle command palette', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);

    slice.setIsCommandPaletteOpen(true);
    expect(set).toHaveBeenCalledWith({ isCommandPaletteOpen: true });
    slice.setIsCommandPaletteOpen(false);
    expect(set).toHaveBeenCalledWith({ isCommandPaletteOpen: false });
  });

  it('should have inlinePanel hidden and no inlineContent by default', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);

    expect(slice.showInlinePanel).toBe(false);
    expect(slice.inlineContent).toBeNull();
    expect(slice.currentScene).toBeNull();
    expect(slice.currentView).toBe('copilot');
  });

  it('should clear inline content to null', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);

    slice.setInlineContent({ type: 'explore', payload: {} });
    slice.setInlineContent(null);
    expect(set).toHaveBeenCalledWith({ inlineContent: null });
  });

  it('should clear slideOver content to null', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createNavigationSlice(set, get, {} as any);

    slice.setSlideOverContent({ type: 'explore', title: 'T' });
    slice.setSlideOverContent(null);
    expect(set).toHaveBeenCalledWith({ slideOverContent: null });
  });
});
