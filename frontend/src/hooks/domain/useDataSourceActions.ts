import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import { IngestionTaskSchema } from '../../schemas'
import { ingestionClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useDataSourceActions(loadProjectData: () => void) {
  const store = useStore()

  return {
    onAddSource: useCallback((config: { name: string; sourceType: string; configJson: string }) => {
      ingestionClient.createTask({ projectId: store.projectID, task: { name: config.name, sourceType: config.sourceType, configJson: config.configJson } })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'createTask'))
    }, [store.projectID, loadProjectData]),
    onRunTask: useCallback((id: string) => {
      ingestionClient.runTask({ projectId: store.projectID, taskId: id })
        .then(() => {
          const poll = () => {
            ingestionClient.getProgress({ projectId: store.projectID, taskId: id })
              .then(() => {
                ingestionClient.listTasks({ projectId: store.projectID }).then((tasksRes) => {
                  const validatedTasks = fromProto(z.array(IngestionTaskSchema), tasksRes.tasks || [])
                  store.setIngestionTasks(validatedTasks)
                  const t = validatedTasks.find((x) => x.id === id)
                  if (t && t.status !== 'completed' && t.status !== 'failed') {
                    setTimeout(poll, 1500)
                  }
                })
              })
              .catch(() => setTimeout(poll, 2000))
          }
          setTimeout(poll, 1000)
        })
        .catch((e: unknown) => handleError(e, 'runTask'))
    }, [store.projectID]),
    onViewLogs: useCallback((id: string) => {
      ingestionClient.getTaskLogs({ projectId: store.projectID, taskId: id })
        .then((res) => store.setTaskLogs(fromProto(z.object({ logs: z.optional(z.string()) }), res).logs || 'Nessun log'))
        .catch((e: unknown) => handleError(e, 'getTaskLogs'))
    }, [store.projectID]),
    onDeleteTask: useCallback((id: string) => {
      ingestionClient.deleteTask({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteTask'))
    }, [store.projectID, loadProjectData]),
  }
}
