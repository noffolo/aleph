interface DetailSlideOverProps {
  data: Record<string, unknown>
  title?: string
  onClose: () => void
}

export function DetailSlideOver({ data, title, onClose }: DetailSlideOverProps) {
  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || 'Dettaglio'}</h3>

      <div className="space-y-3">
        {Object.entries(data).map(([key, value]) => (
          <div key={key}>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">{key}</label>
            <div className="bg-background p-3 rounded-lg border border-border text-sm font-mono">
              {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
            </div>
          </div>
        ))}
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
