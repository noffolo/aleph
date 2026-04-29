import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAppActions } from '../useAppActions';
import { useStore } from '@/store/useStore';
import { projectClient, agentClient, ingestionClient, libraryClient, skillClient, toolClient, nlpClient, authClient, notificationClient, registryClient } from '@/api/factory';

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
    skillClient: { listSkills: mk() },
    toolClient: { listTools: mk(), createTool: mk(), editTool: mk(), deleteTool: mk(), executeTool: mk() },
    nlpClient: { analyzeSentiment: mk() },
    registryClient: { listComponents: mk() },
    sandboxClient: { runSkill: mk(), executeTool: mk() },
    notificationClient: { listChannels: mk(), sendWebhook: mk() },
  };
});

describe('useAppActions', () => {
  const mockStore: Record<string, any> = {
    projectID: 'test-project',
    input: 'hello',
    isStreaming: false,
    selectedAgent: 'agent-1',
    chat: [],
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
    setInlineContent: vi.fn(),
    setCurrentView: vi.fn(),
    setShowInlinePanel: vi.fn(),
    clearChat: vi.fn(),
    addChatMessage: vi.fn(),
    setInput: vi.fn(),
    setIsStreaming: vi.fn(),
    setStreamAbortController: vi.fn(),
    setChat: vi.fn(),
    setPendingConfirmation: vi.fn(),
    setSandboxResult: vi.fn(),
    addToHistory: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (useStore as any).mockReturnValue(mockStore);
    (useStore.getState as any).mockReturnValue(mockStore);
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
});
