import { fromProto } from './validate';
import {
  AgentSchema,
  SkillSchema,
  ToolSchema,
  RegistryComponentSchema,
  ChatMessageSchema,
  PredictionSchema,
  IngestionTaskSchema,
  QueryDataSchema,
  SandboxResultSchema,
  AssetSchema,
  ColumnStatsSchema,
  ContentDataSchema,
  ApiKeySchema,
  ProjectSchema,
  NotificationChannelSchema,
  PendingConfirmationSchema,
} from './index';
import type {
  Agent,
  Skill,
  Tool,
  RegistryComponent,
  ChatMessage,
  Prediction,
  IngestionTask,
  QueryData,
  SandboxResult,
  Asset,
  ColumnStats,
  ContentData,
  ApiKey,
  Project,
  NotificationChannel,
  PendingConfirmation,
} from './index';

export function fromProtoToAgent(proto: unknown): Agent {
  return fromProto(AgentSchema, proto);
}

export function fromProtoToSkill(proto: unknown): Skill {
  return fromProto(SkillSchema, proto);
}

export function fromProtoToTool(proto: unknown): Tool {
  return fromProto(ToolSchema, proto);
}

export function fromProtoToRegistryComponent(proto: unknown): RegistryComponent {
  return fromProto(RegistryComponentSchema, proto);
}

export function fromProtoToChatMessage(proto: unknown): ChatMessage {
  return fromProto(ChatMessageSchema, proto);
}

export function fromProtoToPrediction(proto: unknown): Prediction {
  return fromProto(PredictionSchema, proto);
}

export function fromProtoToIngestionTask(proto: unknown): IngestionTask {
  return fromProto(IngestionTaskSchema, proto);
}

export function fromProtoToQueryData(proto: unknown): QueryData {
  return fromProto(QueryDataSchema, proto);
}

export function fromProtoToSandboxResult(proto: unknown): SandboxResult {
  return fromProto(SandboxResultSchema, proto);
}

export function fromProtoToAsset(proto: unknown): Asset {
  return fromProto(AssetSchema, proto);
}

export function fromProtoToColumnStats(proto: unknown): ColumnStats {
  return fromProto(ColumnStatsSchema, proto);
}

export function fromProtoToContentData(proto: unknown): ContentData {
  return fromProto(ContentDataSchema, proto);
}

export function fromProtoToApiKey(proto: unknown): ApiKey {
  return fromProto(ApiKeySchema, proto);
}

export function fromProtoToProject(proto: unknown): Project {
  return fromProto(ProjectSchema, proto);
}

export function fromProtoToNotificationChannel(proto: unknown): NotificationChannel {
  return fromProto(NotificationChannelSchema, proto);
}

export function fromProtoToPendingConfirmation(proto: unknown): PendingConfirmation {
  return fromProto(PendingConfirmationSchema, proto);
}
