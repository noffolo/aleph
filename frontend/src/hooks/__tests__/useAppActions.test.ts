import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAppActions } from '../useAppActions';
import { useStore } from '@/store/useStore';
import { projectClient, agentClient, ingestionClient, libraryClient, skillClient, toolClient, nlpClient, authClient, notificationClient, registryClient, sandboxClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => {
  const mk = () => vi.fn().mockResolvedValue({});
  return {
    projectClient: { getOntology: mk(), createApiKey: mk(), deleteApiKey: mk() },
    queryClient: { chat: mk(), confirmAction: mk() },
    agentClient: { listAgents: mk(), listModels: mk(), createAgent: mk(), deleteAgent: mk(), updateAgent: mk() },
    ingestionClient: { listTasks: mk() },
    libraryClient: { listAssets: mk(), getAssetContent: mk() },
    authClient: { listApiKeys: mk(), createApiKey: mk(), deleteApiKey: mk() },
    skillClient: { listSkills: mk(), updateSkill: mk() },
    toolClient: { listTools: mk(), createTool: mk(), updateTool: mk(), deleteTool: mk(), executeTool: mk() },
    nlpClient: { analyzeSentiment: mk() },
    registryClient: { listComponents: mk() },
    sandboxClient: { runSkill: mk(), executeTool: mk() },
    notificationClient: { listChannels: mk(), sendWebhook: mk() },
  };
});

describe('useAppActions', () => {
  const mockStore: Record<string, any> = {
    projectID: 'test-project',
    isStreaming: false,
    selectedAgent: 'agent-1',
    messages: [],
    sandboxInput: '{}',
    pendingConfirmation: null,
    setLastError: vi.fn(),
    addToast: vi.fn(),
    setOntologyRaw: vi.fn(),
    setAvailableObjects: vi.fn(),
    setSelectedObject: vi.fn(),
    setAgents: vi.fn(),
    setIngestionTasks: vi.fn(),
    setAssets: vi.fn(),
    setSkills: vi.fn(),
    setTools: vi.fn(),
    setOllamaHealthy: vi.fn(),
    setOllamaModels: vi.fn(),
    setNlpHealthy: vi.fn(),
    setApiKeys: vi.fn(),
    setNotificationChannels: vi.fn(),
    setRegistryComponents: vi.fn(),
    setSlideOverContent: vi.fn(),
    clearMessages: vi.fn(),
    addChatMessage: vi.fn(),
    setIsStreaming: vi.fn(),
    setMessages: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    (useStore as any).mockImplementation((selector?: (state: typeof mockStore) => unknown) => (
      typeof selector === 'function' ? selector(mockStore) : mockStore
    ));
    (useStore.getState as any).mockReturnValue(mockStore);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loadProjectData should fetch and store project data', async () => {
    (projectClient.getOntology as any).mockResolvedValue({ alephDefinition: 'def', objectNames: ['Obj1'] });
    (agentClient.listAgents as any).mockResolvedValue({ agents: [] });
    (ingestionClient.listTasks as any).mockResolvedValue({ tasks: [] });
    (libraryClient.listAssets as any).mockResolvedValue({ assets: [] });
    (skillClient.listSkills as any).mockResolvedValue({ skills: [] });
    (toolClient.listTools as any).mockResolvedValue({ tools: [] });
    (agentClient.listModels as any).mockResolvedValue({ models: ['llama3'] });
    (nlpClient.analyzeSentiment as any).mockResolvedValue({});
    (authClient.listApiKeys as any).mockResolvedValue({ keys: [] });
    (notificationClient.listChannels as any).mockResolvedValue({ channels: [] });
    (registryClient.listComponents as any).mockResolvedValue({ components: [] });

    const { result } = renderHook(() => useAppActions());
    
    await act(async () => {
      await result.current.loadProjectData();
    });

    expect(projectClient.getOntology).toHaveBeenCalledWith(
      expect.objectContaining({ projectId: 'test-project' }),
      expect.any(Object),
    );
    expect(mockStore.setOntologyRaw).toHaveBeenCalledWith('def');
    expect(mockStore.setAvailableObjects).toHaveBeenCalledWith(['Obj1']);
    expect(mockStore.setOllamaModels).toHaveBeenCalledWith(['llama3']);
  });

  it('should handle loadProjectData with errors gracefully', async () => {
    (projectClient.getOntology as any).mockRejectedValue(new Error('API Error'));

    const { result } = renderHook(() => useAppActions());
    
    await act(async () => {
      await result.current.loadProjectData();
    });

    expect(projectClient.getOntology).toHaveBeenCalled();
  });

  describe('JSON.parse error handling (catch block items 2-3)', () => {
    it('onRunSkill passes parsed JSON input to sandboxClient when sandboxInput is valid', async () => {
      mockStore.sandboxInput = '{"mode":"fast","threshold":0.8}';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.runSkill as any).mockResolvedValue({
        result: { stdout: 'ok', stderr: '', exitCode: 0 },
      });

      const { result } = renderHook(() => useAppActions());

      await act(async () => {
        await result.current.onRunSkill('skill-1');
      });

      expect(sandboxClient.runSkill).toHaveBeenCalledWith(
        expect.objectContaining({
          skillId: 'skill-1',
          inputParams: { mode: 'fast', threshold: 0.8 },
        }),
      );
    });

    it('onRunSkill defaults to {} when sandboxInput is malformed (empty catch #2)', async () => {
      mockStore.sandboxInput = '{{broken json}}';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.runSkill as any).mockResolvedValue({
        result: { stdout: 'ok', stderr: '', exitCode: 0 },
      });

      const { result } = renderHook(() => useAppActions());

      await act(async () => {
        await result.current.onRunSkill('skill-1');
      });

      expect(sandboxClient.runSkill).toHaveBeenCalledWith(
        expect.objectContaining({
          inputParams: {},
        }),
      );
    });

    it('onRunSkill does not throw with completely empty sandboxInput', async () => {
      mockStore.sandboxInput = '';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.runSkill as any).mockResolvedValue({
        result: { stdout: 'ok', stderr: '', exitCode: 0 },
      });

      const { result } = renderHook(() => useAppActions());

      await expect(
        act(async () => {
          await result.current.onRunSkill('skill-1');
        }),
      ).resolves.toBeUndefined();
    });

    it('onExecuteTool passes parsed JSON to sandboxClient when valid', async () => {
      mockStore.sandboxInput = '{"tool":"scraper","url":"https://example.com"}';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.executeTool as any).mockResolvedValue({
        result: { stdout: 'done', stderr: '', exitCode: 0 },
      });

      const { result } = renderHook(() => useAppActions());

      await act(async () => {
        await result.current.onExecuteTool('tool-1');
      });

      expect(sandboxClient.executeTool).toHaveBeenCalledWith(
        expect.objectContaining({
          toolId: 'tool-1',
          inputParams: { tool: 'scraper', url: 'https://example.com' },
        }),
      );
    });

    it('onExecuteTool defaults to {} when sandboxInput is malformed (empty catch #3)', async () => {
      mockStore.sandboxInput = 'not json at all';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.executeTool as any).mockResolvedValue({
        result: { stdout: 'ok', stderr: '', exitCode: 0 },
      });

      const { result } = renderHook(() => useAppActions());

      await act(async () => {
        await result.current.onExecuteTool('tool-1');
      });

      expect(sandboxClient.executeTool).toHaveBeenCalledWith(
        expect.objectContaining({
          inputParams: {},
        }),
      );
    });

    it('onExecuteTool does not crash when JSON.parse throws on malformed input', async () => {
      mockStore.sandboxInput = '{broken';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.executeTool as any).mockResolvedValue({
        result: { stdout: 'ok', stderr: '', exitCode: 0 },
      });

      const { result } = renderHook(() => useAppActions());

      await expect(
        act(async () => {
          await result.current.onExecuteTool('tool-1');
        }),
      ).resolves.toBeUndefined();
    });

    it('onRunSkill still handles sandbox API errors after successful JSON.parse', async () => {
      mockStore.sandboxInput = '{"cmd":"test"}';
      (useStore.getState as any).mockReturnValue(mockStore);
      (sandboxClient.runSkill as any).mockRejectedValue(new Error('Sandbox timeout'));

      const { result } = renderHook(() => useAppActions());

      await act(async () => {
        await result.current.onRunSkill('skill-1');
      });

      expect(mockStore.addToast).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'error', context: 'runSkill' }),
      );
    });
  });
});
