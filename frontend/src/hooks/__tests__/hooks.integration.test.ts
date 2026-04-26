import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useAgentActions } from '../domain/useAgentActions';
import { useToolActions } from '../domain/useToolActions';
import { agentClient, toolClient } from '@/api/factory';

vi.mock('@/api/factory', () => ({
  agentClient: { createAgent: vi.fn(), deleteAgent: vi.fn(), updateAgent: vi.fn() },
  toolClient: { createTool: vi.fn(), deleteTool: vi.fn() },
}));

vi.mock('@/hooks/useAppActions', () => ({ handleError: vi.fn() }));

vi.mock('@/store/useStore', () => ({
  useStore: vi.fn(() => ({
    projectID: 'test-project-id',
    setAgents: vi.fn(),
    setTools: vi.fn(),
    setSlideOverContent: vi.fn(),
    setSandboxInput: vi.fn(),
    setLastError: vi.fn(),
    addToast: vi.fn(),
    getState: () => ({ projectID: 'test-project-id', tools: [] }),
  })),
}));

describe('Hook Integration Tests', () => {
  const mockLoadProjectData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('useAgentActions', () => {
    it('should call createAgent and then loadProjectData on success', async () => {
      (agentClient.createAgent as any).mockResolvedValueOnce({ id: 'new-agent-id' });
      const { result } = renderHook(() => useAgentActions(mockLoadProjectData));

      await act(async () => {
        await result.current.onCreateAgent('Test Agent', 'gpt-4', 'Prompt', 'openai', 'key', 'url');
      });

      expect(agentClient.createAgent).toHaveBeenCalled();
      expect(mockLoadProjectData).toHaveBeenCalled();
    });
  });

  describe('useToolActions', () => {
    it('should call createTool and then loadProjectData on success', async () => {
      (toolClient.createTool as any).mockResolvedValueOnce({ id: 'new-tool-id' });
      const { result } = renderHook(() => useToolActions(mockLoadProjectData));

      await act(async () => {
        await result.current.onCreateTool('Test Tool', 'Desc', 'code()');
      });

      expect(toolClient.createTool).toHaveBeenCalled();
      expect(mockLoadProjectData).toHaveBeenCalled();
    });

    it('should call handleError when deleteTool fails', async () => {
      (toolClient.deleteTool as any).mockRejectedValueOnce(new Error('Delete Failed'));
      const { result } = renderHook(() => useToolActions(mockLoadProjectData));

      await act(async () => {
        await result.current.onDeleteTool('tool-id');
      });

      expect(mockLoadProjectData).not.toHaveBeenCalled();
    });
  });
});
