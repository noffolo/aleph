import { z } from "zod";

// ──────────────────────────────────────────────
// ApiKey
// ──────────────────────────────────────────────
export const ApiKeySchema = z.object({
  id: z.coerce.string(),
  label: z.string(),
  key: z.string(),
  createdAt: z.coerce.number(),
});
export type ApiKey = z.infer<typeof ApiKeySchema>;

// ──────────────────────────────────────────────
// Project
// ──────────────────────────────────────────────
export const ProjectSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
});
export type Project = z.infer<typeof ProjectSchema>;

// ──────────────────────────────────────────────
// NotificationChannel
// ──────────────────────────────────────────────
export const NotificationChannelSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  type: z.string(),
  configJson: z.string(),
});
export type NotificationChannel = z.infer<typeof NotificationChannelSchema>;

// ──────────────────────────────────────────────
// RegistryComponent
// ──────────────────────────────────────────────
export const RegistryComponentSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  description: z.string(),
  version: z.string(),
  type: z.string(),
  category: z.string(),
  source: z.string(),
  status: z.string(),
  approvalStatus: z.string(),
  configSchemaJson: z.optional(z.string()),
  executionCommand: z.optional(z.string()),
  dependenciesJson: z.optional(z.string()),
  inputSchemaJson: z.optional(z.string()),
  outputSchemaJson: z.optional(z.string()),
  promptTemplate: z.optional(z.string()),
  toolIdsJson: z.optional(z.string()),
  avgLatencyMs: z.optional(z.number()),
  avgBrierScore: z.optional(z.number()),
  avgCpuUsage: z.optional(z.number()),
  avgMemoryMb: z.optional(z.number()),
  avgExecTimeMs: z.optional(z.number()),
  trustScore: z.optional(z.number()),
  createdByAgentId: z.optional(z.string()),
  creationTimestamp: z.optional(z.string()),
  lastUpdatedTimestamp: z.optional(z.string()),
});
export type RegistryComponent = z.infer<typeof RegistryComponentSchema>;

// ──────────────────────────────────────────────
// ChatMessage
// ──────────────────────────────────────────────
export const ChatMessageSchema = z.object({
  role: z.enum(["user", "assistant", "system"]),
  content: z.string(),
  toolCall: z.optional(z.string()),
  requiresConfirmation: z.optional(z.boolean()),
  createdAt: z.coerce.number(),
});
export type ChatMessage = z.infer<typeof ChatMessageSchema>;

// ──────────────────────────────────────────────
// PendingConfirmation
// ──────────────────────────────────────────────
export const PendingConfirmationSchema = z.object({
  projectId: z.string(),
  agentId: z.string(),
});
export type PendingConfirmation = z.infer<typeof PendingConfirmationSchema>;

// ──────────────────────────────────────────────
// Agent (passthrough — allows [key: string]: unknown)
// ──────────────────────────────────────────────
export const AgentSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  model: z.string(),
  systemPrompt: z.string(),
  provider: z.optional(z.string()),
  apiKey: z.optional(z.string()),
  baseUrl: z.optional(z.string()),
  skillIds: z.optional(z.array(z.string())),
}).passthrough();
export type Agent = z.infer<typeof AgentSchema>;

// ──────────────────────────────────────────────
// Skill (passthrough — allows [key: string]: unknown)
// ──────────────────────────────────────────────
export const SkillSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  description: z.string(),
  toolIds: z.optional(z.array(z.string())),
}).passthrough();
export type Skill = z.infer<typeof SkillSchema>;

// ──────────────────────────────────────────────
// Tool (passthrough — allows [key: string]: unknown)
// ──────────────────────────────────────────────
export const ToolSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  description: z.string(),
  code: z.string(),
}).passthrough();
export type Tool = z.infer<typeof ToolSchema>;

// ──────────────────────────────────────────────
// Prediction
// ──────────────────────────────────────────────
export const PredictionSchema = z.object({
  entityId: z.string(),
  probability: z.number(),
  predictedState: z.string(),
  explanation: z.string(),
});
export type Prediction = z.infer<typeof PredictionSchema>;

// ──────────────────────────────────────────────
// IngestionTask
// ──────────────────────────────────────────────
export const IngestionTaskSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  sourceType: z.string(),
  status: z.string(),
  progress: z.number(),
});
export type IngestionTask = z.infer<typeof IngestionTaskSchema>;

// ──────────────────────────────────────────────
// Row
// ──────────────────────────────────────────────
export const RowSchema = z.object({
  values: z.record(
    z.string(),
    z.union([z.string(), z.number(), z.boolean(), z.null()]),
  ),
});
export type Row = z.infer<typeof RowSchema>;

// ──────────────────────────────────────────────
// QueryData (references RowSchema for rows)
// ──────────────────────────────────────────────
export const QueryDataSchema = z.object({
  columns: z.optional(z.array(z.string())),
  rows: z.optional(z.array(RowSchema)),
  sql: z.optional(z.string()),
});
export type QueryData = z.infer<typeof QueryDataSchema>;

// ──────────────────────────────────────────────
// SandboxResult (passthrough — allows [key: string]: unknown)
// ──────────────────────────────────────────────
export const SandboxResultSchema = z.object({
  exitCode: z.optional(z.number()),
  stdout: z.optional(z.string()),
  stderr: z.optional(z.string()),
  metricsJson: z.optional(z.string()),
}).passthrough();
export type SandboxResult = z.infer<typeof SandboxResultSchema>;

// ──────────────────────────────────────────────
// Asset
// ──────────────────────────────────────────────
export const AssetSchema = z.object({
  id: z.coerce.string(),
  name: z.string(),
  type: z.string(),
  createdAt: z.coerce.number(),
});
export type Asset = z.infer<typeof AssetSchema>;

// ──────────────────────────────────────────────
// ColumnStats (count/uniqueCount/topValues support bigint | number)
// ──────────────────────────────────────────────
const BigIntOrNumber = z.union([z.bigint(), z.number()]);

export const ColumnStatsSchema = z.object({
  columnName: z.string(),
  min: z.string(),
  max: z.string(),
  count: BigIntOrNumber,
  uniqueCount: BigIntOrNumber,
  topValues: z.record(z.string(), BigIntOrNumber),
});
export type ColumnStats = z.infer<typeof ColumnStatsSchema>;

// ──────────────────────────────────────────────
// ContentData (passthrough — allows [key: string]: unknown)
// ──────────────────────────────────────────────
export const ContentDataSchema = z.object({
  name: z.optional(z.string()),
  description: z.optional(z.string()),
  code: z.optional(z.string()),
  id: z.optional(z.coerce.string()),
  toolIds: z.optional(z.array(z.string())),
  exitCode: z.optional(z.number()),
  stdout: z.optional(z.string()),
  stderr: z.optional(z.string()),
  metricsJson: z.optional(z.string()),
  assetId: z.optional(z.string()),
  componentId: z.optional(z.string()),
}).passthrough();
export type ContentData = z.infer<typeof ContentDataSchema>;
