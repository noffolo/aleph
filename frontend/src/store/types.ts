export interface ApiKey {
  id: string
  label: string
  key: string
  createdAt: number
}

export interface Project {
  id: string
  name: string
}

export interface NotificationChannel {
  id: string
  name: string
  type: string
  configJson: string
}

export interface RegistryComponent {
  id: string
  name: string
  description: string
  version: string
  type: string
  category: string
  source: string
  status: string
  approvalStatus: string
  configSchemaJson?: string
  executionCommand?: string
  dependenciesJson?: string
  inputSchemaJson?: string
  outputSchemaJson?: string
  promptTemplate?: string
  toolIdsJson?: string
  avgLatencyMs?: number
  avgBrierScore?: number
  avgCpuUsage?: number
  avgMemoryMb?: number
  avgExecTimeMs?: number
  trustScore?: number
  createdByAgentId?: string
  creationTimestamp?: string
  lastUpdatedTimestamp?: string
}

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCall?: string
  requiresConfirmation?: boolean
  createdAt: number
}

export interface PendingConfirmation {
  projectId: string
  agentId: string
}

export interface Agent {
  id: string
  name: string
  model: string
  systemPrompt: string
  provider?: string
  apiKey?: string
  baseUrl?: string
  skillIds?: string[]
  [key: string]: unknown
}

export interface Skill {
  id: string
  name: string
  description: string
  toolIds?: string[]
  [key: string]: unknown
}

export interface Tool {
  id: string
  name: string
  description: string
  code: string
  [key: string]: unknown
}

export interface Scenario {
  id: string
  name: string
  confidence: number
  signals: { id: string; name: string; strength: number }[]
  assumptions: string[]
  trend: 'up' | 'down' | 'neutral'
  probability: number
  description?: string
}

export interface Prediction {
  entityId: string
  probability: number
  predictedState: string
  explanation: string
}

export interface IngestionTask {
  id: string
  name: string
  sourceType: string
  status: string
  progress: number
}

export interface Row {
  values: Record<string, string | number | boolean | null>
}

export interface QueryData {
  columns?: string[]
  rows?: Row[]
  sql?: string
}

export interface SandboxResult {
  exitCode?: number
  stdout?: string
  stderr?: string
  metricsJson?: string
  [key: string]: unknown
}

export interface Asset {
  id: string
  name: string
  type: string
  createdAt: number
}

export interface ColumnStats {
  columnName: string
  min: string
  max: string
  count: bigint | number
  uniqueCount: bigint | number
  topValues: Record<string, bigint | number>
}

export interface ContentData {
  name?: string
  description?: string
  code?: string
  id?: string
  toolIds?: string[]
  exitCode?: number
  stdout?: string
  stderr?: string
  metricsJson?: string
  assetId?: string
  componentId?: string
  [key: string]: unknown
}

export interface ToolIntel {
  id: string
  name: string
  totalExecutions: number
  avgLatencyMs: number
  errorRate: number
  lastUsed: number
  brierScore: number
  trustScore: number
  execCount: number
  avgDuration: number
  topUsers: string[]
  riskScore: number
  warnings: string[]
  usageFreq: 'high' | 'medium' | 'low'
  recommendations: string[]
  anomalies: { desc: string; severity: 'low' | 'medium' | 'high' }[]
  relatedTools: string[]
}

export interface ToolAnomaly {
  toolId: string
  toolName: string
  anomalyType: 'latency' | 'error_rate' | 'trust_drop'
  severity: 'low' | 'medium' | 'high'
  detectedAt: number
  message: string
}
