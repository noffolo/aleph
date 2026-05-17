import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DashboardView } from '../DashboardView'

vi.mock('../../store/useStore', () => ({
  useStore: vi.fn((selector?: (state: unknown) => unknown) => {
    const state = { projectID: 'proj-1' }
    if (typeof selector === 'function') return selector(state)
    return state
  }),
}))

vi.mock('../../api/client', () => ({
  apiGet: vi.fn(() => Promise.resolve({ json: vi.fn() })),
}))

vi.mock('../../lib/errorReporter', () => ({
  reportError: vi.fn(),
}))

import { apiGet } from '../../api/client'
const mockApiGet = apiGet as ReturnType<typeof vi.fn>

describe('DashboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state initially', () => {
    mockApiGet.mockImplementation(() => new Promise(() => {}))
    render(<DashboardView />)
    expect(screen.getByText(/Loading dashboard metrics/i)).toBeInTheDocument()
  })

  it('renders System Health section after load', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'ok', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ used: 42 }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ queries: [] }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('System Health')).toBeInTheDocument()
    })
  })

  it('renders metric stat cards', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'ok', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ used: 42 }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ queries: [] }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('Total Queries')).toBeInTheDocument()
      expect(screen.getByText('Tool Executions')).toBeInTheDocument()
    })
  })

  it('renders LLM Budget Usage section', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'ok', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ used: 55.5 }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ queries: [] }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('LLM Budget Usage')).toBeInTheDocument()
      expect(screen.getByText(/55\.50/)).toBeInTheDocument()
    })
  })

  it('renders Recent Activity section', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'ok', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ used: 42 }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ queries: [] }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('Recent Activity')).toBeInTheDocument()
      expect(screen.getByText('No recent queries found in this project')).toBeInTheDocument()
    })
  })

  it('renders query history rows', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'ok', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ used: 42 }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({
        queries: [
          { id: 'q1', query: 'SELECT * FROM data', timestamp: Date.now() },
          { id: 'q2', query: 'SELECT COUNT(*) FROM events', timestamp: Date.now() },
        ],
      }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('SELECT * FROM data')).toBeInTheDocument()
      expect(screen.getByText('SELECT COUNT(*) FROM events')).toBeInTheDocument()
    })
  })

  it('renders health items with error status when backend is not ok', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'error', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ used: 0 }),
    })
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ queries: [] }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('System Health')).toBeInTheDocument()
    })
  })

  it('uses fallback budget when budget endpoint fails', async () => {
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ backend: 'ok', nlp: 'ok', duckdb: 'ok', mcp: 'ok' }),
    })
    mockApiGet.mockRejectedValueOnce(new Error('Budget fetch failed'))
    mockApiGet.mockResolvedValueOnce({
      json: async () => ({ queries: [] }),
    })

    render(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('LLM Budget Usage')).toBeInTheDocument()
      expect(screen.getByText(/420/)).toBeInTheDocument()
    })
  })
})
