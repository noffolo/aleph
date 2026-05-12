import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAgentActions } from '../domain/useAgentActions';
import { useStore } from '@/store/useStore';
import { agentClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  agentClient: {
    createAgent: vi.fn(),
    deleteAgent: vi.fn(),
    updateAgent: vi.fn(),
  },
}));

describe('useAgentActions', () => {
  const mockStore: Record<string, any> = {
    projectID: 'test-project',
    agents: [],
    setAgents: vi.fn(),
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

  it('onCreateAgent should call agentClient and then loadProjectData', async () => {
    (agentClient as any).createAgent.mockResolvedValue({});
    
    const { result } = renderHook(() => useAgentActions(mockLoadProjectData));
    
    await act(async () => {
      await result.current.onCreateAgent('Name', 'model', 'prompt', 'provider', 'key', 'url');
    });

    expect(agentClient.createAgent).toHaveBeenCalledWith(expect.objectContaining({
      projectId: 'test-project',
      agent: expect.objectContaining({ name: 'Name' })
    }));
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onDeleteAgent should call agentClient and then loadProjectData', async () => {
    (agentClient as any).deleteAgent.mockResolvedValue({});
    
    const { result } = renderHook(() => useAgentActions(mockLoadProjectData));
    
    await act(async () => {
      await result.current.onDeleteAgent('agent-123');
    });

    expect(agentClient.deleteAgent).toHaveBeenCalledWith({
      projectId: 'test-project',
      id: 'agent-123'
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onUpdateAgent should call agentClient and then loadProjectData', async () => {
    (agentClient as any).updateAgent.mockResolvedValue({});
    
    const { result } = renderHook(() => useAgentActions(mockLoadProjectData));
    
    await act(async () => {
      await result.current.onUpdateAgent({ id: '1', name: 'New' } as any);
    });

    expect(agentClient.updateAgent).toHaveBeenCalledWith(expect.objectContaining({
      projectId: 'test-project'
    }));
    expect(mockLoadProjectData).toHaveBeenCalled();
  });
});
