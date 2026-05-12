import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useOntologyActions } from '../domain/useOntologyActions';
import { useStore } from '@/store/useStore';
import { projectClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  projectClient: {
    emergeOntology: vi.fn(),
    saveOntology: vi.fn(),
  },
}));

// Mock global fetch for version-related functions
const mockFetch = vi.fn();

describe('useOntologyActions', () => {
  const mockStore: Record<string, unknown> = {
    projectID: 'test-project',
    ontologyRaw: 'entities:\n  - Person',
    setOntologyRaw: vi.fn(),
    setLastError: vi.fn(),
    addToast: vi.fn(),
  };

  const mockLoadProjectData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    globalThis.fetch = mockFetch;
    (useStore as any).mockImplementation((selector?: any) => {
      if (typeof selector === 'function') return selector(mockStore);
      return mockStore;
    });
    (useStore.getState as any).mockReturnValue(mockStore);
  });

  it('onEmerge calls emergeOntology and updates store', async () => {
    (projectClient as any).emergeOntology.mockResolvedValue({
      alephDefinition: 'entities:\n  - Company',
    });

    const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onEmerge();
    });

    expect(projectClient.emergeOntology).toHaveBeenCalledWith({
      projectId: 'test-project',
    });
    expect(mockStore.setOntologyRaw).toHaveBeenCalledWith('entities:\n  - Company');
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onEmerge defaults to empty string when alephDefinition is missing', async () => {
    (projectClient as any).emergeOntology.mockResolvedValue({});

    const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onEmerge();
    });

    expect(mockStore.setOntologyRaw).toHaveBeenCalledWith('');
  });

  it('onEmerge handles error via handleError', async () => {
    (projectClient as any).emergeOntology.mockRejectedValue(new Error('Network failure'));

    const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onEmerge();
    });

    expect(mockStore.setLastError).toHaveBeenCalledWith('Network failure');
    expect(mockStore.addToast).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'error', context: 'emergeOntology' }),
    );
  });

  it('onSave saves ontology with current raw text', async () => {
    (projectClient as any).saveOntology.mockResolvedValue({});

    const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onSave();
    });

    expect(projectClient.saveOntology).toHaveBeenCalledWith({
      projectId: 'test-project',
      alephDefinition: 'entities:\n  - Person',
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onSave handles error gracefully', async () => {
    (projectClient as any).saveOntology.mockRejectedValue(new Error('Save failed'));

    const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onSave();
    });

    expect(mockStore.setLastError).toHaveBeenCalledWith('Save failed');
  });

  describe('fetchVersions', () => {
    it('fetches ontology versions successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ versions: [{ id: 'v1' }, { id: 'v2' }] }),
      });

      const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

      await act(async () => {
        await result.current.fetchVersions();
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/ontology/versions?project_id=test-project',
        { credentials: 'include' },
      );
    });

    it('handles fetch error on non-ok response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
      });

      const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

      await act(async () => {
        await result.current.fetchVersions();
      });

      expect(mockStore.setLastError).toHaveBeenCalledWith('Failed to fetch ontology versions');
    });
  });

  describe('acceptVersion', () => {
    it('accepts a version and reloads', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true }); // accept
      mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) }); // fetchVersions

      const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

      await act(async () => {
        await result.current.acceptVersion('v1');
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/ontology/accept',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ version_id: 'v1' }),
        }),
      );
      expect(mockLoadProjectData).toHaveBeenCalled();
    });

    it('handles accept error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
      });

      const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

      await act(async () => {
        await result.current.acceptVersion('bad-version');
      });

      expect(mockStore.setLastError).toHaveBeenCalledWith('Failed to accept ontology version');
    });
  });

  describe('rejectVersion', () => {
    it('rejects a version with reason', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true });

      const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));

      await act(async () => {
        await result.current.rejectVersion('v1', 'Invalid entities');
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/ontology/reject',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ version_id: 'v1', reason: 'Invalid entities' }),
        }),
      );
    });
  });

  it('exposes setOntologyRaw from store', () => {
    const { result } = renderHook(() => useOntologyActions(mockLoadProjectData));
    expect(result.current.setOntologyRaw).toBe(mockStore.setOntologyRaw);
  });
});
