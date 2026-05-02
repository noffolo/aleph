import { useCallback, useEffect, useRef } from 'react'
import { useStore } from '../../store/useStore'
import { IngestionTaskSchema } from '../../schemas'
import { ingestionClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useDataSourceActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    return () => {
      abortRef.current?.abort()
    }
  }, [])

  return {
    onAddSource: useCallback((config: { name: string; sourceType: string; configJson: string }) => {
      ingestionClient.createTask({ projectId: projectID, task: { name: config.name, sourceType: config.sourceType, configJson: config.configJson } })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'createTask'))
    }, [projectID, loadProjectData]),
    onRunTask: useCallback((id: string) => {
      abortRef.current?.abort()
      const ac = new AbortController()
      abortRef.current = ac

      ingestionClient.runTask({ projectId: projectID, taskId: id })
        .then(() => {
          const poll = () => {
            if (ac.signal.aborted) return

            ingestionClient.getProgress({ projectId: projectID, taskId: id })
              .then(() => {
                if (ac.signal.aborted) return
                ingestionClient.listTasks({ projectId: projectID }).then((tasksRes) => {
                  if (ac.signal.aborted) return
                  const validatedTasks = fromProto(z.array(IngestionTaskSchema), tasksRes.tasks || [])
                  useStore.getState().setIngestionTasks(validatedTasks)
                  const t = validatedTasks.find((x) => x.id === id)
                  if (t && t.status !== 'completed' && t.status !== 'failed') {
                    setTimeout(poll, 1500)
                  }
                })
              })
              .catch(() => {
                if (!ac.signal.aborted) setTimeout(poll, 2000)
              })
          }
          setTimeout(poll, 1000)
        })
        .catch((e: unknown) => handleError(e, 'runTask'))
    }, [projectID]),
    onViewLogs: useCallback((id: string) => {
      ingestionClient.getTaskLogs({ projectId: projectID, taskId: id })
        .then((res) => useStore.getState().setTaskLogs(fromProto(z.object({ logs: z.optional(z.string()) }), res).logs || 'Nessun log'))
        .catch((e: unknown) => handleError(e, 'getTaskLogs'))
    }, [projectID]),
    onDeleteTask: useCallback((id: string) => {
      ingestionClient.deleteTask({ projectId: projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteTask'))
    }, [projectID, loadProjectData]),
  }
}
