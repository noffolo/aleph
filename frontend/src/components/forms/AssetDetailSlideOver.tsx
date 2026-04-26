import { useStore } from '../../store/useStore'
import { useViewActions } from '../../hooks/useViewActions'

interface AssetDetailSlideOverProps {
  assetId: string
  title?: string
  onClose: () => void
}

export function AssetDetailSlideOver({ assetId, title, onClose }: AssetDetailSlideOverProps) {
  const store = useStore()
  const actions = useViewActions()
  const asset = store.assets.find(a => a.id === assetId)

  if (!asset) return null

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{asset.name || title}</h3>
      <p className="text-textMuted">Asset Type: {asset.type}</p>

      <div className="space-y-4">
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Preview</label>
          <div className="bg-background p-4 rounded-lg border border-border text-sm">
            Mostra contenuto asset...
          </div>
        </div>

        <div className="flex gap-3">
          <button
            onClick={() => actions.libraryActions.onGetAssetContent(assetId)}
            className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
          >
            Vedi Contenuto
          </button>
          <button
            onClick={() => actions.libraryActions.onGeneratePdf(assetId)}
            className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
          >
            Genera PDF
          </button>
        </div>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          onClick={onClose}
          className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
        >
          Chiudi
        </button>
      </div>
    </div>
  )
}
