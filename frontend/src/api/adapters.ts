import type { Agent, Skill, Tool } from '../store/types'

export function fromProtoAgent(raw: Record<string, unknown>): Agent {
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? ''),
    model: String(raw.model ?? ''),
    systemPrompt: String(raw.systemPrompt ?? ''),
    provider: raw.provider != null ? String(raw.provider) : undefined,
    apiKey: raw.apiKey != null ? String(raw.apiKey) : undefined,
    baseUrl: raw.baseUrl != null ? String(raw.baseUrl) : undefined,
    skillIds: Array.isArray(raw.skillIds) ? raw.skillIds.map(String) : undefined,
  }
}

export function fromProtoSkill(raw: Record<string, unknown>): Skill {
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? ''),
    description: String(raw.description ?? ''),
    toolIds: Array.isArray(raw.toolIds) ? raw.toolIds.map(String) : undefined,
  }
}

export function fromProtoTool(raw: Record<string, unknown>): Tool {
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? ''),
    description: String(raw.description ?? ''),
    code: String(raw.code ?? ''),
  }
}

export function fromProtoAgentList(raw: unknown): Agent[] {
  return Array.isArray(raw) ? raw.map((item) => fromProtoAgent(item as Record<string, unknown>)) : []
}

export function fromProtoSkillList(raw: unknown): Skill[] {
  return Array.isArray(raw) ? raw.map((item) => fromProtoSkill(item as Record<string, unknown>)) : []
}

export function fromProtoToolList(raw: unknown): Tool[] {
  return Array.isArray(raw) ? raw.map((item) => fromProtoTool(item as Record<string, unknown>)) : []
}
