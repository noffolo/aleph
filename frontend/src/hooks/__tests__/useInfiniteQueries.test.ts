import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import { useInfiniteAgents, useInfiniteSkills, useInfiniteTools } from '../useInfiniteQueries'

vi.mock('../../api/factory', () => ({
  agentClient: { listAgents: vi.fn() },
  skillClient: { listSkills: vi.fn() },
  toolClient: { listTools: vi.fn() },
}))

vi.mock('../../schemas', () => ({
  AgentSchema: { parse: (a: any) => a },
  SkillSchema: { parse: (s: any) => s },
  ToolSchema: { parse: (t: any) => t },
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useInfiniteQueries', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('useInfiniteAgents', () => {
    it('is disabled when projectId is empty', () => {
      const { result } = renderHook(() => useInfiniteAgents(''), { wrapper: createWrapper() })
      expect(result.current.isPending).toBe(true)
    })

    it('fetches agents when projectId is provided', async () => {
      const { agentClient } = await import('../../api/factory')
      ;(agentClient.listAgents as ReturnType<typeof vi.fn>).mockResolvedValue({
        agents: [{ id: 'a1', name: 'Agent 1' }],
        nextCursor: '',
      })
      const { result } = renderHook(() => useInfiniteAgents('p1'), { wrapper: createWrapper() })
      await waitFor(() => { expect(result.current.isSuccess).toBe(true) })
      expect(result.current.data?.pages[0].items).toEqual([{ id: 'a1', name: 'Agent 1' }])
    })
  })

  describe('useInfiniteSkills', () => {
    it('is disabled when projectId is empty', () => {
      const { result } = renderHook(() => useInfiniteSkills(''), { wrapper: createWrapper() })
      expect(result.current.isPending).toBe(true)
    })

    it('fetches skills when projectId is provided', async () => {
      const { skillClient } = await import('../../api/factory')
      ;(skillClient.listSkills as ReturnType<typeof vi.fn>).mockResolvedValue({
        skills: [{ id: 's1', name: 'Skill 1' }],
        nextCursor: '',
      })
      const { result } = renderHook(() => useInfiniteSkills('p1'), { wrapper: createWrapper() })
      await waitFor(() => { expect(result.current.isSuccess).toBe(true) })
      expect(result.current.data?.pages[0].items).toEqual([{ id: 's1', name: 'Skill 1' }])
    })
  })

  describe('useInfiniteTools', () => {
    it('is disabled when projectId is empty', () => {
      const { result } = renderHook(() => useInfiniteTools(''), { wrapper: createWrapper() })
      expect(result.current.isPending).toBe(true)
    })

    it('fetches tools when projectId is provided', async () => {
      const { toolClient } = await import('../../api/factory')
      ;(toolClient.listTools as ReturnType<typeof vi.fn>).mockResolvedValue({
        tools: [{ id: 't1', name: 'Tool 1' }],
        nextCursor: '',
      })
      const { result } = renderHook(() => useInfiniteTools('p1'), { wrapper: createWrapper() })
      await waitFor(() => { expect(result.current.isSuccess).toBe(true) })
      expect(result.current.data?.pages[0].items).toEqual([{ id: 't1', name: 'Tool 1' }])
    })
  })
})
