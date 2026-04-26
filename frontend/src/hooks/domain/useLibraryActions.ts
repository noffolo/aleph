import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Asset } from '../../store/types'
import { libraryClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useLibraryActions(loadProjectData: () => void) {
  const store = useStore()

  return {
    onViewAsset: useCallback((id: string) => {
      store.setSelectedAssetId(id)
      libraryClient.getAssetContent({ projectId: store.projectID, assetId: id })
        .then((res) => store.setSelectedAssetContent(fromProto(z.object({ content: z.optional(z.string()) }), res).content || ''))
        .catch(() => store.setSelectedAssetContent('Errore nel caricamento'))
    }, [store.projectID]),
    onDeleteAsset: useCallback((id: string) => {
      libraryClient.deleteAsset({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteAsset'))
    }, [store.projectID, loadProjectData]),
    selectedAssetContent: store.selectedAssetContent,
    setSelectedAssetContent: store.setSelectedAssetContent,
    selectedAssetName: store.assets.find((a: Asset) => a.id === store.selectedAssetId)?.name,
    onGetAssetContent: useCallback(async (assetId: string) => {
      const res = await libraryClient.getAssetContent({ projectId: store.projectID, assetId })
      return fromProto(z.object({ content: z.optional(z.string()) }), res).content || ''
    }, [store.projectID]),
    onGeneratePdf: useCallback(async (assetId: string) => {
      const res = fromProto(z.object({ pdfData: z.unknown(), filename: z.unknown() }), await libraryClient.generatePdf({ projectId: store.projectID, assetId }))
      return { pdfData: res.pdfData as Uint8Array, filename: res.filename as string }
    }, [store.projectID]),
    onUploadAsset: useCallback(async (filename: string, content: Uint8Array) => {
      await libraryClient.uploadAsset({ projectId: store.projectID, filename, content })
      loadProjectData()
    }, [store.projectID, loadProjectData]),
    selectedAssetId: store.selectedAssetId,
  }
}
