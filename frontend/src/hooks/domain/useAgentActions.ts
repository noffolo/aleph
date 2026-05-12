import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Agent } from '../../store/types'
import { agentClient } from '../../api/factory'
import { handleError } from '../useAppActions'

export function useAgentActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)

  return {
    onCreateAgent: useCallback((name: string, model: string, systemPrompt: string, provider: string, apiKey: string, baseUrl: string) => {
      const store = useStore.getState()
      agentClient.createAgent({ projectId: projectID, agent: { name, model, systemPrompt, provider: provider || 'ollama', apiKey: apiKey || '', baseUrl: baseUrl || '' } })
        .then((res: any) => {
          const newAgent = res.agent || { id: res.id || `agent-${Date.now()}`, name, model, systemPrompt, provider: provider || 'ollama' }
          const current = useStore.getState()
          current.setAgents([newAgent, ...current.agents])
          loadProjectData()
        })
        .catch((e: unknown) => {
          handleError(e, 'createAgent')
        })
    }, [projectID, loadProjectData]),

    onDeleteAgent: useCallback((id: string) => {
      const store = useStore.getState()
      const previousAgents = [...store.agents]
      store.setAgents(store.agents.filter(a => a.id !== id))
      agentClient.deleteAgent({ projectId: projectID, id })
        .then(() => {
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.setAgents(previousAgents)
          handleError(e, 'deleteAgent')
        })
    }, [projectID, loadProjectData]),

    onUpdateAgent: useCallback((agent: Agent) => {
      const store = useStore.getState()
      const previousAgents = [...store.agents]
      store.setAgents(store.agents.map(a => a.id === agent.id ? agent : a))
      agentClient.updateAgent({ projectId: projectID, agent })
        .then(() => {
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.setAgents(previousAgents)
          handleError(e, 'updateAgent')
        })
    }, [projectID, loadProjectData]),
  }
}
