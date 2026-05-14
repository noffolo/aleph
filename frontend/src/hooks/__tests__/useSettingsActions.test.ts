import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSettingsActions } from '../domain/useSettingsActions';
import { useStore } from '@/store/useStore';
import { authClient, notificationClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  authClient: {
    createApiKey: vi.fn(),
    listApiKeys: vi.fn(),
    deleteApiKey: vi.fn(),
  },
  notificationClient: {
    sendWebhook: vi.fn(),
  },
}));

describe('useSettingsActions', () => {
  const mockStore: Record<string, unknown> = {
    projectID: 'test-project',
    apiKeys: [],
    setApiKeys: vi.fn(),
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

  describe('onCreateApiKey', () => {
    it('creates a key and refreshes the list', async () => {
      (authClient as any).createApiKey.mockResolvedValue({});
      (authClient as any).listApiKeys.mockResolvedValue({
        keys: [{ id: 'k1', label: 'My Key', key: 'secret-123', createdAt: 1234567890 }],
      });

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onCreateApiKey('My Key');
      });

      expect(authClient.createApiKey).toHaveBeenCalledWith({
        projectId: 'test-project',
        label: 'My Key',
      });
      expect(mockStore.setApiKeys).toHaveBeenCalled();
    });

    it('handles creation error', async () => {
      (authClient as any).createApiKey.mockRejectedValue(new Error('Limit reached'));

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onCreateApiKey('Too Many');
      });

      expect(mockStore.addToast).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'error', context: 'createApiKey' }),
      );
    });
  });

  describe('onDeleteApiKey', () => {
    it('deletes a key and refreshes the list', async () => {
      (authClient as any).deleteApiKey.mockResolvedValue({});
      (authClient as any).listApiKeys.mockResolvedValue({ keys: [] });

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onDeleteApiKey('k1');
      });

      expect(authClient.deleteApiKey).toHaveBeenCalledWith({
        projectId: 'test-project',
        id: 'k1',
      });
      expect(mockStore.setApiKeys).toHaveBeenCalled();
    });

    it('handles deletion error', async () => {
      (authClient as any).deleteApiKey.mockRejectedValue(new Error('Not authorized'));

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onDeleteApiKey('protected-key');
      });

      expect(mockStore.addToast).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'error', context: 'deleteApiKey' }),
      );
    });
  });

  describe('onSendWebhook', () => {
    it('sends webhook and shows success toast', async () => {
      (notificationClient as any).sendWebhook.mockResolvedValue({
        success: true,
      });

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onSendWebhook('https://hook.example.com', '{"data":1}', 'secret123');
      });

      expect(notificationClient.sendWebhook).toHaveBeenCalledWith({
        url: 'https://hook.example.com',
        payloadJson: '{"data":1}',
        secret: 'secret123',
      });
      expect(mockStore.addToast).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'success', context: 'sendWebhook' }),
      );
    });

    it('handles webhook failure with error from response', async () => {
      (notificationClient as any).sendWebhook.mockResolvedValue({
        success: false,
        error: 'Invalid URL format',
      });

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onSendWebhook('bad-url', '{}', '');
      });

      expect(mockStore.addToast).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'error', context: 'sendWebhook' }),
      );
    });

    it('handles network error when sending webhook', async () => {
      (notificationClient as any).sendWebhook.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useSettingsActions());

      await act(async () => {
        await result.current.onSendWebhook('https://example.com', '{}', 'secret');
      });

      expect(mockStore.addToast).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'error', context: 'sendWebhook' }),
      );
    });
  });
});
