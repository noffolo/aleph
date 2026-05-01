import { useStore } from '../../store/useStore'
import type { RegistryComponent } from '../../store/types'
import { t } from '../../i18n'

interface ComponentDetailSlideOverProps {
  componentId: string
  title?: string
  onClose: () => void
}

export function ComponentDetailSlideOver({ componentId, title, onClose }: ComponentDetailSlideOverProps) {
  const registryComponents = useStore(s => s.registryComponents)
  const component = registryComponents.find(c => c.id === componentId)

  if (!component) return null

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{component.name || title}</h3>
      <p className="text-textMuted">{component.description || 'Nessuna descrizione'}</p>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Tipo</div>
          <div className="text-sm">{component.type}</div>
        </div>
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Categoria</div>
          <div className="text-sm">{component.category}</div>
        </div>
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Sorgente</div>
          <div className="text-sm">{component.source}</div>
        </div>
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Stato</div>
          <div className="text-sm">{component.status}</div>
        </div>
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Approvazione</div>
          <div className="text-sm">{component.approvalStatus}</div>
        </div>
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Versione</div>
          <div className="text-sm">{component.version}</div>
        </div>
      </div>

      {component.promptTemplate && (
        <div>
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-2">Prompt Template</div>
          <div className="p-3 bg-background rounded-lg border border-border text-xs font-mono whitespace-pre-wrap">{component.promptTemplate}</div>
        </div>
      )}

      <div className="flex gap-3 pt-2">
        <button
          onClick={onClose}
          className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
        >
          {t('slideOver.close')}
        </button>
      </div>
    </div>
  )
}
