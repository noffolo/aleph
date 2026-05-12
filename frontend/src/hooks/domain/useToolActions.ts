import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Tool } from '../../store/types'
import { toolClient } from '../../api/factory'
import { handleError } from '../useAppActions'

export function useToolActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)

  return {
    onCreateTool: useCallback((name: string, description: string, code: string) => {
      const store = useStore.getState()
      const optimistic: Tool = { id: `pending-${Date.now()}`, name, description, code }
      store.setTools([optimistic, ...store.tools])
      toolClient.createTool({ projectId: projectID, tool: { name, description, code } })
        .then(() => {
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.setTools(s.tools.filter(t => t.id !== optimistic.id))
          handleError(e, 'createTool')
        })
    }, [projectID, loadProjectData]),

    onUpdateTool: useCallback((tool: Tool) => {
      const store = useStore.getState()
      const previousTools = [...store.tools]
      store.setTools(store.tools.map(t => t.id === tool.id ? tool : t))
      toolClient.updateTool({ projectId: projectID, tool })
        .then(() => {
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.setTools(previousTools)
          handleError(e, 'updateTool')
        })
    }, [projectID, loadProjectData]),

    onEditTool: useCallback((tool: Tool) => {
      useStore.getState().setSlideOverContent({ type: 'tool', title: tool.name, data: tool })
    }, []),

    onDeleteTool: useCallback((id: string) => {
      const store = useStore.getState()
      const previousTools = [...store.tools]
      store.setTools(store.tools.filter(t => t.id !== id))
      toolClient.deleteTool({ projectId: projectID, id })
        .then(() => {
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.setTools(previousTools)
          handleError(e, 'deleteTool')
        })
    }, [projectID, loadProjectData]),

    onExecuteTool: useCallback((id: string) => {
      const tool = useStore.getState().tools.find((t: Tool) => t.id === id)
      if (tool) useStore.getState().setSlideOverContent({ type: 'tool', title: tool.name, data: tool })
      useStore.getState().setSandboxInput('{}')
    }, []),
  }
}
