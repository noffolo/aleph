import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useComponentActions } from '../domain/useComponentActions';
import { useStore } from '@/store/useStore';
import { registryClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  registryClient: {
    updateComponentStatus: vi.fn(),
    registerComponent: vi.fn(),
    getComponent: vi.fn(),
    listComponents: vi.fn(),
  },
}));

describe('useComponentActions', () => {
  const mockStore: Record<string, unknown> = {
    projectID: 'test-project',
    registryComponents: [],
    setRegistryComponents: vi.fn(),
    setLastError: vi.fn(),
    addToast: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    (useStore as any).mockImplementation((selector?: any) => {
      if (typeof selector === 'function') return selector(mockStore);
      return mockStore;
    });
    (useStore.getState as any).mockReturnValue(mockStore);
  });

  it('onUpdateComponentStatus updates status and refreshes list', async () => {
    (registryClient as any).updateComponentStatus.mockResolvedValue({});
    (registryClient as any).listComponents.mockResolvedValue({
      components: [
        { id: 'c1', name: 'Component A', description: 'desc', status: 'active', type: 'tool', category: 'finance', source: 'internal', approvalStatus: 'approved', version: '1.0' },
      ],
    });

    const { result } = renderHook(() => useComponentActions());

    await act(async () => {
      await result.current.onUpdateComponentStatus('c1', 'active');
    });

    expect(registryClient.updateComponentStatus).toHaveBeenCalledWith({
      id: 'c1',
      status: 'active',
    });
    expect(mockStore.setRegistryComponents).toHaveBeenCalled();
  });

  it('onUpdateComponentStatus handles error', async () => {
    (registryClient as any).updateComponentStatus.mockRejectedValue(new Error('Component not found'));

    const { result } = renderHook(() => useComponentActions());

    await act(async () => {
      await result.current.onUpdateComponentStatus('missing', 'active');
    });

    expect(mockStore.setLastError).toHaveBeenCalledWith('Component not found');
  });

  it('onRegisterComponent registers and refreshes list', async () => {
    (registryClient as any).registerComponent.mockResolvedValue({});
    (registryClient as any).listComponents.mockResolvedValue({ components: [] });

    const component = {
      id: 'new-comp',
      name: 'New Component',
      description: 'A new component',
      version: '1.0',
      type: 'tool',
      category: 'finance',
      source: 'internal',
      status: 'active',
      approvalStatus: 'pending',
      creationTimestamp: '2026-01-01',
      lastUpdatedTimestamp: '2026-01-01',
    };

    const { result } = renderHook(() => useComponentActions());

    await act(async () => {
      await result.current.onRegisterComponent(component as any);
    });

    // Should strip creationTimestamp and lastUpdatedTimestamp
    expect(registryClient.registerComponent).toHaveBeenCalledWith({
      metadata: {
        id: 'new-comp',
        name: 'New Component',
        description: 'A new component',
        version: '1.0',
        type: 'tool',
        category: 'finance',
        source: 'internal',
        status: 'active',
        approvalStatus: 'pending',
      },
    });
    expect(mockStore.setRegistryComponents).toHaveBeenCalled();
  });

  it('onGetComponent returns component metadata on success', async () => {
    (registryClient as any).getComponent.mockResolvedValue({
      metadata: {
        id: 'c1',
        name: 'Found Component',
        version: '1.0',
        type: 'tool',
        category: 'finance',
        source: 'internal',
        status: 'active',
        approvalStatus: 'approved',
        description: 'desc',
      },
    });

    const { result } = renderHook(() => useComponentActions());

    let component: unknown;
    await act(async () => {
      component = await result.current.onGetComponent('c1');
    });

    expect(registryClient.getComponent).toHaveBeenCalledWith({ id: 'c1' });
    expect(component).toEqual(expect.objectContaining({ id: 'c1', name: 'Found Component' }));
  });

  it('onGetComponent returns null when metadata is missing', async () => {
    (registryClient as any).getComponent.mockResolvedValue({});

    const { result } = renderHook(() => useComponentActions());

    let component: unknown;
    await act(async () => {
      component = await result.current.onGetComponent('c1');
    });

    expect(component).toBeNull();
  });

  it('onGetComponent returns null and logs error on failure', async () => {
    (registryClient as any).getComponent.mockRejectedValue(new Error('Not found'));

    const { result } = renderHook(() => useComponentActions());

    let component: unknown;
    await act(async () => {
      component = await result.current.onGetComponent('missing');
    });

    expect(component).toBeNull();
    expect(mockStore.setLastError).toHaveBeenCalledWith('Not found');
  });

  it('onRegisterComponent handles error gracefully', async () => {
    (registryClient as any).registerComponent.mockRejectedValue(new Error('Validation error'));

    const { result } = renderHook(() => useComponentActions());

    await act(async () => {
      await result.current.onRegisterComponent({ id: 'bad' } as any);
    });

    expect(mockStore.setLastError).toHaveBeenCalledWith('Validation error');
  });
});
