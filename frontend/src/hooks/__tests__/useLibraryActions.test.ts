import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useLibraryActions } from '../domain/useLibraryActions';
import { useStore } from '@/store/useStore';
import { libraryClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  libraryClient: {
    getAssetContent: vi.fn(),
    deleteAsset: vi.fn(),
    generatePdf: vi.fn(),
    uploadAsset: vi.fn(),
  },
}));

describe('useLibraryActions', () => {
  const mockStore: Record<string, unknown> = {
    projectID: 'test-project',
    assets: [
      { id: 'asset-1', name: 'Report 2026.pdf', type: 'pdf', createdAt: 1715000000 },
      { id: 'asset-2', name: 'Dataset CSV', type: 'csv', createdAt: 1715000100 },
    ],
    selectedAssetId: 'asset-1',
    selectedAssetContent: '',
    setSelectedAssetContent: vi.fn(),
    setSelectedAssetId: vi.fn(),
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

  it('onViewAsset selects asset and loads content', async () => {
    (libraryClient as any).getAssetContent.mockResolvedValue({
      content: '# Report Content',
    });

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onViewAsset('asset-1');
    });

    expect(mockStore.setSelectedAssetId).toHaveBeenCalledWith('asset-1');
    expect(libraryClient.getAssetContent).toHaveBeenCalledWith({
      projectId: 'test-project',
      assetId: 'asset-1',
    });
    expect(mockStore.setSelectedAssetContent).toHaveBeenCalledWith('# Report Content');
  });

  it('onViewAsset shows error message when content fails to load', async () => {
    (libraryClient as any).getAssetContent.mockRejectedValue(new Error('Not found'));

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onViewAsset('asset-1');
    });

    expect(mockStore.setSelectedAssetContent).toHaveBeenCalledWith('Errore nel caricamento');
  });

  it('onDeleteAsset deletes and reloads project data', async () => {
    (libraryClient as any).deleteAsset.mockResolvedValue({});

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onDeleteAsset('asset-1');
    });

    expect(libraryClient.deleteAsset).toHaveBeenCalledWith({
      projectId: 'test-project',
      id: 'asset-1',
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onDeleteAsset handles error via handleError', async () => {
    (libraryClient as any).deleteAsset.mockRejectedValue(new Error('Permission denied'));

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onDeleteAsset('asset-1');
    });

    expect(mockStore.addToast).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'error', context: 'deleteAsset' }),
    );
  });

  it('exposes selectedAssetContent from store', () => {
    mockStore.selectedAssetContent = 'cached content';

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    expect(result.current.selectedAssetContent).toBe('cached content');
  });

  it('exposes selectedAssetName derived from assets and selectedAssetId', () => {
    // mockStore.assets has asset-1 = "Report 2026.pdf"
    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    expect(result.current.selectedAssetName).toBe('Report 2026.pdf');
  });

  it('onGetAssetContent fetches and returns content for a specific asset', async () => {
    (libraryClient as any).getAssetContent.mockResolvedValue({
      content: 'Special content',
    });

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    const content = await act(async () => {
      return await result.current.onGetAssetContent('asset-2');
    });

    expect(libraryClient.getAssetContent).toHaveBeenCalledWith({
      projectId: 'test-project',
      assetId: 'asset-2',
    });
    expect(content).toBe('Special content');
  });

  it('onGetAssetContent returns empty string when content is missing', async () => {
    (libraryClient as any).getAssetContent.mockResolvedValue({});

    const { result: hook } = renderHook(() => useLibraryActions(mockLoadProjectData));

    const content = await act(async () => {
      return await hook.current.onGetAssetContent('asset-1');
    });

    expect(content).toBe('');
  });

  it('onGeneratePdf fetches PDF data', async () => {
    const mockPdfData = new Uint8Array([1, 2, 3]);
    (libraryClient as any).generatePdf.mockResolvedValue({
      pdfData: mockPdfData,
      filename: 'report.pdf',
    });

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    let output: unknown;
    await act(async () => {
      output = await result.current.onGeneratePdf('asset-1');
    });

    expect(libraryClient.generatePdf).toHaveBeenCalledWith({
      projectId: 'test-project',
      assetId: 'asset-1',
    });
    expect(output).toEqual({ pdfData: mockPdfData, filename: 'report.pdf' });
  });

  it('onUploadAsset uploads content and reloads project data', async () => {
    (libraryClient as any).uploadAsset.mockResolvedValue({});

    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    const content = new Uint8Array([4, 5, 6]);
    await act(async () => {
      await result.current.onUploadAsset('new-file.csv', content);
    });

    expect(libraryClient.uploadAsset).toHaveBeenCalledWith({
      projectId: 'test-project',
      filename: 'new-file.csv',
      content,
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('exposes selectedAssetId from store', () => {
    const { result } = renderHook(() => useLibraryActions(mockLoadProjectData));

    expect(result.current.selectedAssetId).toBe('asset-1');
  });
});
