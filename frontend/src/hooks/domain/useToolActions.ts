import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Tool } from '../../store/types'
import { ToolSchema } from '../../schemas'
import { toolClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'

export function useToolActions(loadProjectData: () => void) {
  const store = useStore()

  return {
    onCreateTool: useCallback((name: string, description: string, code: string) => {
      toolClient.createTool({ projectId: store.projectID, tool: { name, description, code } })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'createTool'))
    }, [store.projectID, loadProjectData]),
    onEditTool: useCallback((tool: Tool) => {
      store.setSlideOverContent({ type: 'tool', title: tool.name, data: tool })
    }, []),
    onDeleteTool: useCallback((id: string) => {
      toolClient.deleteTool({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteTool'))
    }, [store.projectID, loadProjectData]),
    onExecuteTool: useCallback((id: string) => {
      const tool = useStore.getState().tools.find((t: Tool) => t.id === id)
      if (tool) useStore.getState().setSlideOverContent({ type: 'tool', title: tool.name, data: tool })
      store.setSandboxInput('{}')
    }, []),
  }
}
