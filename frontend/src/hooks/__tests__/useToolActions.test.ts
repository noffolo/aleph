import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useToolActions } from '../domain/useToolActions';
import { useStore } from '@/store/useStore';
import { toolClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  toolClient: {
    createTool: vi.fn(),
    updateTool: vi.fn(),
    deleteTool: vi.fn(),
  },
}));

describe('useToolActions', () => {
  const mockStore = {
    projectID: 'test-project',
    setSlideOverContent: vi.fn(),
    setSandboxInput: vi.fn(),
    tools: [{ id: 'tool-1', name: 'Test Tool' }],
  };

  const mockLoadProjectData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    (useStore as any).mockImplementation((selector?: any) => {
      if (typeof selector === 'function') return selector(mockStore)
      return mockStore
    });
    (useStore.getState as any).mockReturnValue(mockStore);
  });

  it('onCreateTool should call toolClient and loadProjectData', async () => {
    (toolClient as any).createTool.mockResolvedValue({});
    const { result } = renderHook(() => useToolActions(mockLoadProjectData));
    
    await act(async () => {
      await result.current.onCreateTool('Name', 'Desc', 'code');
    });

    expect(toolClient.createTool).toHaveBeenCalledWith(expect.objectContaining({
      projectId: 'test-project',
      tool: { name: 'Name', description: 'Desc', code: 'code' }
    }));
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onUpdateTool should call toolClient and loadProjectData', async () => {
    (toolClient as any).updateTool.mockResolvedValue({});
    const { result } = renderHook(() => useToolActions(mockLoadProjectData));
    const tool = { id: 'tool-1', name: 'Updated', description: 'Desc', code: 'code' } as any;

    await act(async () => {
      await result.current.onUpdateTool(tool);
    });

    expect(toolClient.updateTool).toHaveBeenCalledWith(expect.objectContaining({
      projectId: 'test-project',
      tool,
    }));
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onEditTool should set slide over content', () => {
    const { result } = renderHook(() => useToolActions(mockLoadProjectData));
    const tool = { id: 'tool-1', name: 'Test Tool' } as any;
    
    result.current.onEditTool(tool);
    
    expect(mockStore.setSlideOverContent).toHaveBeenCalledWith({
      type: 'tool',
      title: 'Test Tool',
      data: tool
    });
  });

  it('onExecuteTool should set slide over content and sandbox input', () => {
    const { result } = renderHook(() => useToolActions(mockLoadProjectData));
    
    result.current.onExecuteTool('tool-1');
    
    expect(mockStore.setSlideOverContent).toHaveBeenCalledWith({
      type: 'tool',
      title: 'Test Tool',
      data: expect.objectContaining({ id: 'tool-1' })
    });
    expect(mockStore.setSandboxInput).toHaveBeenCalledWith('{}');
  });
});
