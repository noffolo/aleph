import { useState } from 'react'
import { useStore } from '../../store/useStore'
import { useViewActions } from '../../hooks/useViewActions'

interface DataSourceFormSlideOverProps {
  title?: string
}

export function DataSourceFormSlideOver({ title }: DataSourceFormSlideOverProps) {
  const store = useStore()
  const actions = useViewActions()
  const [name, setName] = useState('')
  const [sourceType, setSourceType] = useState('csv')
  const [configJson, setConfigJson] = useState('{}')

  const handleSubmit = () => {
    if (!name.trim()) {
      alert('Il nome è obbligatorio')
      return
    }

    try {
      JSON.parse(configJson)
    } catch {
      alert('Config JSON non valido')
      return
    }

    actions.dataSourcesActions.onAddSource({ name, sourceType, configJson })
    store.setSlideOverContent(null)
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || 'Nuova Sorgente Dati'}</h3>

      <div className="space-y-3">
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder="Es: Dati CSV clienti"
          />
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tipo Sorgente</label>
          <select
            value={sourceType}
            onChange={(e) => setSourceType(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
          >
            <option value="csv">CSV</option>
            <option value="api">API REST</option>
            <option value="database">Database</option>
            <option value="json">JSON File</option>
            <option value="xml">XML</option>
          </select>
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Configurazione JSON</label>
          <textarea
            value={configJson}
            onChange={(e) => setConfigJson(e.target.value)}
            rows={6}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={`{\n  "url": "https://...",\n  "format": "csv",\n  "columns": []\n}`}
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
          Crea Sorgente
        </button>
      </div>
    </div>
  )
}
