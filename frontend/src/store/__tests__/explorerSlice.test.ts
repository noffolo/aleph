import { describe, it, expect } from 'vitest';
import { createExplorerSlice } from '../explorerSlice';

const createMockSet = () => {
  let state: any = {};
  return (update: any) => {
    if (typeof update === 'function') {
      state = update(state);
    } else {
      state = { ...state, ...update };
    }
    return state;
  };
};

describe('explorerSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    expect(slice.searchQuery).toBe('');
    expect(slice.selectedObject).toBe('');
    expect(slice.activeView).toBe('table');
    expect(slice.isExplorerLoading).toBe(false);
    expect(slice.globalSearchResults).toBeNull();
  });

  it('should update search query', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    slice.setSearchQuery('test query');
    expect(set({ searchQuery: 'test query' })).toMatchObject({ searchQuery: 'test query' });
  });

  it('should update selected object', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    slice.setSelectedObject('table:users');
    expect(set({ selectedObject: 'table:users' })).toMatchObject({ selectedObject: 'table:users' });
  });

  it('should update active view', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    slice.setActiveView('chart');
    expect(set({ activeView: 'chart' })).toMatchObject({ activeView: 'chart' });
  });

  it('should update isExplorerLoading', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    slice.setIsExplorerLoading(true);
    expect(set({ isExplorerLoading: true })).toMatchObject({ isExplorerLoading: true });
  });

  it('should update globalSearchResults', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    const results = { items: [{ id: '1', name: 'test' }], total: 1 };
    slice.setGlobalSearchResults(results);
    expect(set({ globalSearchResults: results })).toMatchObject({ globalSearchResults: results });
  });

  it('should set globalSearchResults to null', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    slice.setGlobalSearchResults(null);
    expect(set({ globalSearchResults: null })).toMatchObject({ globalSearchResults: null });
  });

  it('should reset explorer to defaults', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createExplorerSlice(set, get, {} as any);

    // Set non-default values first
    slice.setSearchQuery('some query');
    slice.setSelectedObject('some-obj');
    slice.setActiveView('chart');
    slice.setIsExplorerLoading(true);
    slice.setGlobalSearchResults({ items: [{ id: '1' }], total: 1 });

    // Now reset
    slice.resetExplorer();

    expect(set({
      searchQuery: '',
      selectedObject: '',
      activeView: 'table',
      isExplorerLoading: false,
      globalSearchResults: null,
    })).toMatchObject({
      searchQuery: '',
      selectedObject: '',
      activeView: 'table',
      isExplorerLoading: false,
      globalSearchResults: null,
    });
  });
});
