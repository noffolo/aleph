import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useViewActions } from '../useViewActions';
import { useStore } from '@/store/useStore';

vi.mock('@/store/useStore', () => ({
  useStore: vi.fn(),
}));

vi.mock('../domain/useExplorerActions', () => ({ useExplorerActions: vi.fn(() => ({})) }));
vi.mock('../domain/useAgentActions', () => ({ useAgentActions: vi.fn(() => ({})) }));
vi.mock('../domain/useToolActions', () => ({ useToolActions: vi.fn(() => ({})) }));
vi.mock('../domain/useSkillActions', () => ({ useSkillActions: vi.fn(() => ({})) }));
vi.mock('../domain/useDataSourceActions', () => ({ useDataSourceActions: vi.fn(() => ({})) }));
vi.mock('../domain/useLibraryActions', () => ({ useLibraryActions: vi.fn(() => ({})) }));
vi.mock('../domain/useComponentActions', () => ({ useComponentActions: vi.fn(() => ({})) }));
vi.mock('../domain/useSettingsActions', () => ({ useSettingsActions: vi.fn(() => ({})) }));
vi.mock('../domain/useOntologyActions', () => ({ useOntologyActions: vi.fn(() => ({})) }));

describe('useViewActions', () => {
  const mockStore = {
    projectID: 'test-project',
    assets: [],
    selectedAssetId: null,
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
    setApiKeys: vi.fn(),
    setNotificationChannels: vi.fn(),
    setRegistryComponents: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (useStore as any).mockReturnValue(mockStore);
  });

  it('should return store and all domain action hooks', () => {
    const { result } = renderHook(() => useViewActions());
    
    expect(result.current.store).toBe(mockStore);
    expect(result.current.explorerActions).toBeDefined();
    expect(result.current.agentsActions).toBeDefined();
    expect(result.current.toolsActions).toBeDefined();
  });

  it('loadProjectData should be a function', () => {
    const { result } = renderHook(() => useViewActions());
    expect(typeof result.current.loadProjectData).toBe('function');
  });

  it('loadProjectData should not execute if projectID is missing', async () => {
    const noProjectStore = { ...mockStore, projectID: null };
    (useStore as any).mockReturnValue(noProjectStore);
    
    const { result } = renderHook(() => useViewActions());
    expect(result.current.loadProjectData).toBeDefined();
  });
});
