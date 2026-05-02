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
      store.setPendingCrud('createTool')
      const optimistic: Tool = { id: `pending-${Date.now()}`, name, description, code }
      store.setTools([optimistic, ...store.tools])
      toolClient.createTool({ projectId: projectID, tool: { name, description, code } })
        .then(() => {
          useStore.getState().clearPendingCrud('createTool')
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.clearPendingCrud('createTool')
          s.setTools(s.tools.filter(t => t.id !== optimistic.id))
          handleError(e, 'createTool')
        })
    }, [projectID, loadProjectData]),

    onUpdateTool: useCallback((tool: Tool) => {
      const store = useStore.getState()
      const previousTools = [...store.tools]
      store.setPendingCrud(`update-${tool.id}`)
      store.setTools(store.tools.map(t => t.id === tool.id ? tool : t))
      toolClient.updateTool({ projectId: projectID, tool })
        .then(() => {
          useStore.getState().clearPendingCrud(`update-${tool.id}`)
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.clearPendingCrud(`update-${tool.id}`)
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
      store.setPendingCrud(`delete-${id}`)
      store.setTools(store.tools.filter(t => t.id !== id))
      toolClient.deleteTool({ projectId: projectID, id })
        .then(() => {
          useStore.getState().clearPendingCrud(`delete-${id}`)
          loadProjectData()
        })
        .catch((e: unknown) => {
          const s = useStore.getState()
          s.clearPendingCrud(`delete-${id}`)
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
