import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useExplorerActions } from '../domain/useExplorerActions';
import { useStore } from '@/store/useStore';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

describe('useExplorerActions', () => {
  const mockStore: Record<string, unknown> = {
    availableObjects: ['Object1', 'Object2'],
    searchQuery: '',
    selectedObject: null,
    activeView: 'graph',
    selectedRow: null,
    setSelectedObject: vi.fn(),
    setSearchQuery: vi.fn(),
    setActiveView: vi.fn(),
    setSelectedRow: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (useStore as any).mockImplementation((selector?: any) => {
      if (typeof selector === 'function') return selector(mockStore);
      return mockStore;
    });
  });

  it('returns setSelectedObject from store', () => {
    const { result } = renderHook(() => useExplorerActions());
    expect(result.current.setSelectedObject).toBe(mockStore.setSelectedObject);
  });

  it('returns setSearchQuery from store', () => {
    const { result } = renderHook(() => useExplorerActions());
    expect(result.current.setSearchQuery).toBe(mockStore.setSearchQuery);
  });

  it('returns setActiveView from store', () => {
    const { result } = renderHook(() => useExplorerActions());
    expect(result.current.setActiveView).toBe(mockStore.setActiveView);
  });

  it('onRowClick calls setSelectedRow from store', () => {
    const { result } = renderHook(() => useExplorerActions());
    const row = { values: { id: '123', name: 'Test' } };

    result.current.onRowClick(row);

    expect(mockStore.setSelectedRow).toHaveBeenCalledWith(row);
  });

  it('setSearchQuery can be called to update search', () => {
    const { result } = renderHook(() => useExplorerActions());

    result.current.setSearchQuery('test query');

    expect(mockStore.setSearchQuery).toHaveBeenCalledWith('test query');
  });

  it('setSelectedObject can be called to select an object', () => {
    const { result } = renderHook(() => useExplorerActions());
    const objName = 'Object1';

    result.current.setSelectedObject(objName);

    expect(mockStore.setSelectedObject).toHaveBeenCalledWith(objName);
  });

  it('setActiveView can change view mode', () => {
    const { result } = renderHook(() => useExplorerActions());

    result.current.setActiveView('table');

    expect(mockStore.setActiveView).toHaveBeenCalledWith('table');
  });
});
