import { useState } from 'react'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useToolActions } from '../../hooks/domain/useToolActions'
import type { Tool } from '../../store/types'

interface ToolFormSlideOverProps {
  tool?: Tool
  title?: string
}

export function ToolFormSlideOver({ tool, title }: ToolFormSlideOverProps) {
  const store = useStore()
  const { loadProjectData } = useAppActions()
  const { onCreateTool } = useToolActions(loadProjectData)
  const isEdit = tool && tool.id
  const [name, setName] = useState(tool?.name || '')
  const [description, setDescription] = useState(tool?.description || '')
  const [code, setCode] = useState(tool?.code || '')

  const handleSubmit = () => {
    if (!name.trim()) {
      alert('Il nome è obbligatorio')
      return
    }

    if (isEdit && tool?.id) {
      alert('Update tool non ancora implementato')
      store.setSlideOverContent(null)
    } else {
      onCreateTool(name, description, code)
      store.setSlideOverContent(null)
    }
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? 'Modifica Tool' : 'Nuovo Tool')}</h3>

      <div className="space-y-3">
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder="Es: Analizzatore CSV"
          />
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder="Descrivi cosa fa questo tool..."
          />
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Codice</label>
          <textarea
            value={code}
            onChange={(e) => setCode(e.target.value)}
            rows={8}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder="// Implementazione del tool..."
          />
        </div>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          onClick={() => store.setSlideOverContent(null)}
          className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
        >
          Annulla
        </button>
        <button
          onClick={handleSubmit}
          className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
        >
          {isEdit ? 'Aggiorna Tool' : 'Crea Tool'}
        </button>
      </div>
    </div>
  )
}
