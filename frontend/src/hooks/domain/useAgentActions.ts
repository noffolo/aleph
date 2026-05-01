import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Agent } from '../../store/types'
import { AgentSchema } from '../../schemas'
import { agentClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'

export function useAgentActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)

  return {
    onCreateAgent: useCallback((name: string, model: string, systemPrompt: string, provider: string, apiKey: string, baseUrl: string) => {
      agentClient.createAgent({ projectId: projectID, agent: { name, model, systemPrompt, provider: provider || 'ollama', apiKey: apiKey || '', baseUrl: baseUrl || '' } })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'createAgent'))
    }, [projectID, loadProjectData]),
    onDeleteAgent: useCallback((id: string) => {
      agentClient.deleteAgent({ projectId: projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteAgent'))
    }, [projectID, loadProjectData]),
    onUpdateAgent: useCallback((agent: Agent) => {
      agentClient.updateAgent({ projectId: projectID, agent })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'updateAgent'))
    }, [projectID, loadProjectData]),
  }
}
