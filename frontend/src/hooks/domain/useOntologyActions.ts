import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import { projectClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useOntologyActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)
  const setOntologyRaw = useStore(s => s.setOntologyRaw)
  const ontologyRaw = useStore(s => s.ontologyRaw)

  const fetchVersions = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/ontology/versions?project_id=${projectID}`, {
        credentials: 'include',
      })
      if (!res.ok) throw new Error('Failed to fetch ontology versions')
      const data = await res.json()
      useStore.getState().setOntologyVersions(data.versions || [])
    } catch (e: unknown) {
      handleError(e, 'fetchVersions')
    }
  }, [projectID])

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
    setOntologyRaw,
    onEmerge: useCallback(() => {
      projectClient.emergeOntology({ projectId: projectID })
        .then((res) => { useStore.getState().setOntologyRaw(fromProto(z.object({ alephDefinition: z.optional(z.string()) }), res).alephDefinition || ''); loadProjectData() })
        .catch((e: unknown) => handleError(e, 'emergeOntology'))
    }, [projectID, loadProjectData]),
    onSave: useCallback(() => {
      projectClient.saveOntology({ projectId: projectID, alephDefinition: ontologyRaw })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'saveOntology'))
    }, [projectID, ontologyRaw, loadProjectData]),
    fetchVersions,
    acceptVersion,
    rejectVersion,
  }
}
