import { createPromiseClient } from "@connectrpc/connect";
import { transport } from "./client"; 

import { RegistryService } from "./proto/aleph/v1/registry_connect";
import { SandboxService } from "./proto/aleph/v1/sandbox_connect";
import { QueryService, ProjectService, AgentService, IngestionService, LibraryService, AuthService, SkillService, ToolService } from "./proto/aleph/v1/query_connect";
import { NLPService } from "./proto/aleph/nlp/v1/nlp_connect";
import { NotificationService } from "./proto/aleph/v1/notification_connect";

export const registryClient = createPromiseClient(RegistryService, transport);
export const sandboxClient = createPromiseClient(SandboxService, transport);
export const queryClient = createPromiseClient(QueryService, transport);
export const projectClient = createPromiseClient(ProjectService, transport);
export const agentClient = createPromiseClient(AgentService, transport);
export const ingestionClient = createPromiseClient(IngestionService, transport);
export const libraryClient = createPromiseClient(LibraryService, transport);
export const authClient = createPromiseClient(AuthService, transport);
export const skillClient = createPromiseClient(SkillService, transport);
export const toolClient = createPromiseClient(ToolService, transport);
export const nlpClient = createPromiseClient(NLPService, transport);
export const notificationClient = createPromiseClient(NotificationService, transport);
