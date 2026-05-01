import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Asset } from '../../store/types'
import { libraryClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useLibraryActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)
  const selectedAssetContent = useStore(s => s.selectedAssetContent)
  const setSelectedAssetContent = useStore(s => s.setSelectedAssetContent)
  const selectedAssetId = useStore(s => s.selectedAssetId)
  const selectedAssetName = useStore(s => s.assets.find((a: Asset) => a.id === s.selectedAssetId)?.name)

  return {
    onViewAsset: useCallback((id: string) => {
      useStore.getState().setSelectedAssetId(id)
      libraryClient.getAssetContent({ projectId: projectID, assetId: id })
        .then((res) => useStore.getState().setSelectedAssetContent(fromProto(z.object({ content: z.optional(z.string()) }), res).content || ''))
        .catch(() => useStore.getState().setSelectedAssetContent('Errore nel caricamento'))
    }, [projectID]),
    onDeleteAsset: useCallback((id: string) => {
      libraryClient.deleteAsset({ projectId: projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteAsset'))
    }, [projectID, loadProjectData]),
    selectedAssetContent,
    setSelectedAssetContent,
    selectedAssetName,
    onGetAssetContent: useCallback(async (assetId: string) => {
      const res = await libraryClient.getAssetContent({ projectId: projectID, assetId })
      return fromProto(z.object({ content: z.optional(z.string()) }), res).content || ''
    }, [projectID]),
    onGeneratePdf: useCallback(async (assetId: string) => {
      const res = fromProto(z.object({ pdfData: z.unknown(), filename: z.unknown() }), await libraryClient.generatePdf({ projectId: projectID, assetId }))
      return { pdfData: res.pdfData as Uint8Array, filename: res.filename as string }
    }, [projectID]),
    onUploadAsset: useCallback(async (filename: string, content: Uint8Array) => {
      await libraryClient.uploadAsset({ projectId: projectID, filename, content })
      loadProjectData()
    }, [projectID, loadProjectData]),
    selectedAssetId,
  }
}
