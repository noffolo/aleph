import { useInfiniteQuery } from '@tanstack/react-query'
import { agentClient, skillClient, toolClient } from '../api/factory'
import { AgentSchema, SkillSchema, ToolSchema } from '../schemas'
import type { Agent, Skill, Tool } from '../store/types'

const DEFAULT_PAGE_SIZE = 25
const MAX_PAGE_SIZE = 100

interface PaginatedResponse<T> {
  items: T[]
  nextCursor: string
}

async function fetchAgentsPage(projectId: string, cursor: string): Promise<PaginatedResponse<Agent>> {
  const res = await agentClient.listAgents({
    projectId,
    after: cursor,
    limit: DEFAULT_PAGE_SIZE,
  })
  return {
    items: (res.agents || []).map(a => AgentSchema.parse(a)),
    nextCursor: res.nextCursor || '',
  }
}

async function fetchSkillsPage(projectId: string, cursor: string): Promise<PaginatedResponse<Skill>> {
  const res = await skillClient.listSkills({
    projectId,
    after: cursor,
    limit: DEFAULT_PAGE_SIZE,
  })
  return {
    items: (res.skills || []).map(s => SkillSchema.parse(s)),
    nextCursor: res.nextCursor || '',
  }
}

async function fetchToolsPage(_projectId: string, cursor: string): Promise<PaginatedResponse<Tool>> {
  const res = await toolClient.listTools({
    after: cursor,
    limit: DEFAULT_PAGE_SIZE,
  })
  return {
    items: (res.tools || []).map(t => ToolSchema.parse(t)),
    nextCursor: res.nextCursor || '',
  }
}

export function useInfiniteAgents(projectId: string) {
  return useInfiniteQuery({
    queryKey: ['agents', projectId],
    queryFn: ({ pageParam }) => fetchAgentsPage(projectId, pageParam as string),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.nextCursor || undefined,
    enabled: !!projectId,
    staleTime: 30_000,
  })
}

export function useInfiniteSkills(projectId: string) {
  return useInfiniteQuery({
    queryKey: ['skills', projectId],
    queryFn: ({ pageParam }) => fetchSkillsPage(projectId, pageParam as string),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.nextCursor || undefined,
    enabled: !!projectId,
    staleTime: 30_000,
  })
}

export function useInfiniteTools(projectId: string) {
  return useInfiniteQuery({
    queryKey: ['tools', projectId],
    queryFn: ({ pageParam }) => fetchToolsPage(projectId, pageParam as string),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.nextCursor || undefined,
    enabled: !!projectId,
    staleTime: 30_000,
  })
}
