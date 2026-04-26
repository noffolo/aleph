import { describe, it, expect, vi } from 'vitest';
import { createUISlice } from '../uiSlice';

const createMockSet = () => {
  let state: any = { toastMessages: [] };
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

describe('uiSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createUISlice(set, get, {} as any);
    
    expect(slice.showOnboarding).toBe(true);
    expect(slice.showWizard).toBe(false);
    expect(slice.showGuide).toBe(false);
    expect(slice.isExplorerLoading).toBe(false);
    expect(slice.confirmDialog.isOpen).toBe(false);
    expect(slice.toastMessages).toEqual([]);
  });

  it('should show/hide confirm dialog', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createUISlice(set, get, {} as any);
    
    slice.showConfirmDialog('Are you sure?', 'Confirm', () => {});
    expect(set).toHaveBeenCalledTimes(1);
    
    slice.hideConfirmDialog();
    expect(set).toHaveBeenCalledTimes(2);
  });

  it('should manage toast messages', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createUISlice(set, get, {} as any);
    
    slice.addToast({ message: 'Success!', type: 'success' });
    expect(set).toHaveBeenCalled();
    
    slice.removeToast('toast-1');
    expect(set).toHaveBeenCalledTimes(2);
  });

  it('should reset UI', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createUISlice(set, get, {} as any);
    
    slice.resetUI();
    expect(set).toHaveBeenCalled();
  });
});
