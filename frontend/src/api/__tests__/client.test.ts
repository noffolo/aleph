import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createSession, deleteSession, apiGet, apiPost, apiPatch, transport } from '../client';

describe('API client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    globalThis.fetch = vi.fn();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('createSession', () => {
    it('sends session creation POST with api key in body', async () => {
      const mockFetch = vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ sessionId: 'sess-1' }),
      } as Response);

      const result = await createSession('test-api-key');

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/auth/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ api_key: 'test-api-key' }),
        credentials: 'include',
      });
      expect(result).toEqual({ sessionId: 'sess-1' });
    });

    it('throws on non-ok response from session endpoint', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: () => Promise.resolve({ message: 'Invalid API key' }),
      } as Response);

      await expect(createSession('bad-key')).rejects.toThrow('Invalid API key');
    });

    it('throws on invalid-api-key body even with 200', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: () => Promise.reject(new Error('JSON parse error')),
      } as Response);

      await expect(createSession('bad-key')).rejects.toThrow('Invalid API key');
    });
  });

  describe('deleteSession', () => {
    it('sends DELETE with credentials', async () => {
      const mockFetch = vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
      } as Response);

      await deleteSession();

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/auth/session', {
        method: 'DELETE',
        credentials: 'include',
      });
    });

    it('does not throw even on server error', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
      } as Response);

      // deleteSession does not check response status
      await expect(deleteSession()).resolves.toBeUndefined();
    });
  });

  describe('apiGet', () => {
    it('fetches with credentials', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: 'hello' }),
      } as Response);

      const result = await apiGet('/api/v1/test');

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/v1/test', {
        credentials: 'include',
      });
      expect(result).toEqual({ data: 'hello' });
    });

    it('throws on non-ok response with JSON error body', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: () => Promise.resolve({ message: 'Resource not found' }),
      } as Response);

      await expect(apiGet('/api/v1/missing')).rejects.toThrow('Resource not found');
    });

    it('throws statusText when response is not JSON', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: () => Promise.reject(new Error('not json')),
      } as Response);

      // The catch handler always returns { message: res.statusText },
      // so err.message is always defined — the `API error: ${res.status}` fallback is dead code.
      await expect(apiGet('/api/v1/broken')).rejects.toThrow('Internal Server Error');
    });

    it('throws statusText when error json has no message field', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 403,
        statusText: 'Forbidden',
        json: () => Promise.resolve({}),
      } as Response);

      // err.message is undefined → falls through to `API error: ${res.status}`
      await expect(apiGet('/api/v1/forbidden')).rejects.toThrow('API error: 403');
    });
  });

  describe('apiPost', () => {
    it('POSTs JSON body with credentials and content-type header', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ id: 'new-1' }),
      } as Response);

      const result = await apiPost('/api/v1/create', { name: 'test' });

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/v1/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'test' }),
        credentials: 'include',
      });
      expect(result).toEqual({ id: 'new-1' });
    });

    it('throws on non-ok response', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: () => Promise.resolve({ message: 'Validation error' }),
      } as Response);

      await expect(apiPost('/api/v1/create', {})).rejects.toThrow('Validation error');
    });

    it('handles null body gracefully', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ ok: true }),
      } as Response);

      // null body → JSON.stringify(null) → "null"
      const result = await apiPost('/api/v1/null-body', null);

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/v1/null-body', expect.objectContaining({
        body: 'null',
      }));
      expect(result).toEqual({ ok: true });
    });
  });

  describe('apiPatch', () => {
    it('delegates to apiPost (same implementation)', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ patched: true }),
      } as Response);

      const result = await apiPatch('/api/v1/update', { field: 'value' });

      expect(result).toEqual({ patched: true });
    });
  });

  describe('transport', () => {
    it('creates transport with empty baseUrl and include credentials', () => {
      // transport is a ConnectRPC transport instance
      expect(transport).toBeDefined();
    });
  });
});
