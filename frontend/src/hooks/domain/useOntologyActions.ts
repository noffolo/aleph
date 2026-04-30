import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import { projectClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useOntologyActions(loadProjectData: () => void) {
  const store = useStore()

  const fetchVersions = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/ontology/versions?project_id=${store.projectID}`, {
        credentials: 'include',
      })
      if (!res.ok) throw new Error('Failed to fetch ontology versions')
      const data = await res.json()
      store.setOntologyVersions(data.versions || [])
    } catch (e: unknown) {
      handleError(e, 'fetchVersions')
    }
  }, [store.projectID, store.setOntologyVersions])

  const acceptVersion = useCallback(async (versionId: string) => {
    try {
      const res = await fetch('/api/v1/ontology/accept', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ version_id: versionId }),
        credentials: 'include',
      })
      if (!res.ok) throw new Error('Failed to accept ontology version')
      await loadProjectData()
      await fetchVersions()
    } catch (e: unknown) {
      handleError(e, 'acceptVersion')
    }
  }, [fetchVersions, loadProjectData])

  const rejectVersion = useCallback(async (versionId: string, reason: string) => {
    try {
      const res = await fetch('/api/v1/ontology/reject', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ version_id: versionId, reason }),
        credentials: 'include',
      })
      if (!res.ok) throw new Error('Failed to reject ontology version')
      await fetchVersions()
    } catch (e: unknown) {
      handleError(e, 'rejectVersion')
    }
  }, [fetchVersions])

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
    fetchVersions,
    acceptVersion,
    rejectVersion,
  }
}
