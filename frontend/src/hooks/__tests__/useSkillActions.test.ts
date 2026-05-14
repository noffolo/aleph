import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSkillActions } from '../domain/useSkillActions';
import { useStore } from '@/store/useStore';
import { skillClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  skillClient: {
    createSkill: vi.fn(),
    updateSkill: vi.fn(),
    deleteSkill: vi.fn(),
  },
}));

describe('useSkillActions', () => {
  const mockStore: Record<string, unknown> = {
    projectID: 'test-project',
    skills: [
      { id: 'skill-1', name: 'Analisi Sentiment', description: 'Analyze sentiment', toolIds: ['t1'] },
    ],
    setSlideOverContent: vi.fn(),
    setSandboxInput: vi.fn(),
    setLastError: vi.fn(),
    addToast: vi.fn(),
  };

  const mockLoadProjectData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    (useStore as any).mockImplementation((selector?: any) => {
      if (typeof selector === 'function') return selector(mockStore);
      return mockStore;
    });
    (useStore.getState as any).mockReturnValue(mockStore);
  });

  it('onCreateSkill calls skillClient and reloads', async () => {
    (skillClient as any).createSkill.mockResolvedValue({});

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onCreateSkill('My Skill', 'Description', ['t1', 't2']);
    });

    expect(skillClient.createSkill).toHaveBeenCalledWith({
      projectId: 'test-project',
      skill: { name: 'My Skill', description: 'Description', toolIds: ['t1', 't2'] },
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onCreateSkill handles error', async () => {
    (skillClient as any).createSkill.mockRejectedValue(new Error('Duplicate name'));

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onCreateSkill('Bad', 'Desc', []);
    });

    expect(mockStore.addToast).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'error', context: 'createSkill' }),
    );
  });

  it('onUpdateSkill calls skillClient and reloads', async () => {
    (skillClient as any).updateSkill.mockResolvedValue({});
    const skill = { id: 'skill-1', name: 'Updated', description: 'New desc', toolIds: ['t3'] };

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onUpdateSkill(skill as any);
    });

    expect(skillClient.updateSkill).toHaveBeenCalledWith({
      projectId: 'test-project',
      skill,
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onDeleteSkill calls skillClient and reloads', async () => {
    (skillClient as any).deleteSkill.mockResolvedValue({});

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onDeleteSkill('skill-1');
    });

    expect(skillClient.deleteSkill).toHaveBeenCalledWith({
      projectId: 'test-project',
      id: 'skill-1',
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onViewSkillDetail sets slide over content', () => {
    const skill = { id: 'skill-1', name: 'Analisi Sentiment', description: 'desc' };

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    result.current.onViewSkillDetail(skill as any);

    expect(mockStore.setSlideOverContent).toHaveBeenCalledWith({
      type: 'skill',
      title: 'Analisi Sentiment',
      data: skill,
    });
  });

  it('onRunSkill sets slide over and sandbox input for existing skill', () => {
    (useStore.getState as any).mockReturnValue(mockStore);

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    result.current.onRunSkill('skill-1');

    expect(mockStore.setSlideOverContent).toHaveBeenCalledWith({
      type: 'skill',
      title: 'Analisi Sentiment',
      data: expect.objectContaining({ id: 'skill-1' }),
    });
    expect(mockStore.setSandboxInput).toHaveBeenCalledWith('{}');
  });

  it('onRunSkill does not set slide over for non-existent skill', () => {
    (useStore.getState as any).mockReturnValue(mockStore);

    const { result } = renderHook(() => useSkillActions(mockLoadProjectData));

    result.current.onRunSkill('nonexistent');

    // Slide over should not be called because find returns undefined
    expect(mockStore.setSandboxInput).toHaveBeenCalledWith('{}');
  });
});
