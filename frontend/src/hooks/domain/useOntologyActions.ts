import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import { projectClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useOntologyActions(loadProjectData: () => void) {
  const store = useStore()

  return {
    setOntologyRaw: store.setOntologyRaw,
    onEmerge: useCallback(() => {
      projectClient.emergeOntology({ projectId: store.projectID })
        .then((res) => { store.setOntologyRaw(fromProto(z.object({ alephDefinition: z.optional(z.string()) }), res).alephDefinition || ''); loadProjectData() })
        .catch((e: unknown) => handleError(e, 'emergeOntology'))
    }, [store.projectID, loadProjectData]),
    onSave: useCallback(() => {
      projectClient.saveOntology({ projectId: store.projectID, alephDefinition: store.ontologyRaw })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'saveOntology'))
    }, [store.projectID, store.ontologyRaw, loadProjectData]),
  }
}
