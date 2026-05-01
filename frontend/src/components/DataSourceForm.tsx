import React, { useState, useEffect } from 'react'
import { Upload, Globe, Database, FileText, Code, Link } from 'lucide-react'
import { DataSourceFormSchema } from '../schemas'
import type { DataSourceFormData } from '../schemas'

interface DataSourceFormProps {
  onSave: (data: DataSourceFormData) => void
  onCancel: () => void
  title?: string
}

type Mode = 'file' | 'api' | 'db'

export function DataSourceForm({ onSave, onCancel, title }: DataSourceFormProps) {
  const [mode, setMode] = useState<Mode>('file')
  const [errors, setErrors] = useState<Partial<Record<keyof DataSourceFormData, string>>>({})
  
  const [formData, setFormData] = useState<DataSourceFormData>({
    name: '',
    sourceType: 'csv',
    configJson: JSON.stringify({ path: '', delimiter: ',', hasHeader: true }, null, 2)
  })

  useEffect(() => {
    let defaultConfig = '{}'
    let defaultType = 'csv'

    if (mode === 'file') {
      defaultType = 'csv'
      defaultConfig = JSON.stringify({ path: '', delimiter: ',', hasHeader: true }, null, 2)
    } else if (mode === 'api') {
      defaultType = 'api'
      defaultConfig = JSON.stringify({ url: '', method: 'GET', headers: {}, format: 'json' }, null, 2)
    } else if (mode === 'db') {
      defaultType = 'database'
      defaultConfig = JSON.stringify({ connectionString: '', query: '', type: 'postgresql' }, null, 2)
    }

    setFormData(prev => ({ ...prev, sourceType: defaultType, configJson: defaultConfig }))
  }, [mode])

  const validate = (): boolean => {
    const result = DataSourceFormSchema.safeParse(formData)
    if (!result.success) {
      const newErrors: Partial<Record<keyof DataSourceFormData, string>> = {}
      for (const issue of result.error.issues) {
        const field = issue.path[0] as keyof DataSourceFormData
        if (!newErrors[field]) {
          newErrors[field] = issue.message
        }
      }
      // Additional cross-field validation for mode-specific rules
      if (mode === 'api') {
        try {
          const config = JSON.parse(formData.configJson)
          if (config.url && !/^https?:\/\/.+/.test(config.url)) {
            newErrors.configJson = "URL in config must start with http:// or https://"
          }
        } catch {}
      }
      if (mode === 'db') {
        try {
          const config = JSON.parse(formData.configJson)
          if (!config.connectionString || !config.connectionString.trim()) {
            newErrors.configJson = "connectionString is required in config JSON"
          }
        } catch {}
      }
      setErrors(newErrors)
      return false
    }
    setErrors({})
    return true
  }

  const handleSubmit = () => {
    if (validate()) {
      onSave(formData)
    }
  }

  const handleFileFormatChange = (format: 'csv' | 'json' | 'xml') => {
    const templates = {
      csv: JSON.stringify({ path: '', delimiter: ',', hasHeader: true }, null, 2),
      json: JSON.stringify({ path: '', rootPath: '' }, null, 2),
      xml: JSON.stringify({ path: '', recordPath: '' }, null, 2),
    }
    setFormData(prev => ({ 
      ...prev, 
      sourceType: format, 
      configJson: templates[format] 
    }))
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || 'Nuova Sorgente Dati'}</h3>
      
      <div className="space-y-3">
        <div>
          <label htmlFor="datasource-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            id="datasource-name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder="Es: Dati Clienti Q3"
          />
          {errors.name && <p className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>

        <div>
          <label htmlFor="datasource-mode" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Modalità Sorgente</label>
          <div className="grid grid-cols-3 gap-2">
            <button
              onClick={() => setMode('file')}
              className={`flex flex-col items-center justify-center p-3 rounded-lg border transition-all ${
                mode === 'file' ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-background text-textDim hover:bg-surface-alt'
              }`}
            >
              <Upload size={20} className="mb-2" />
              <span className="text-[10px] font-bold uppercase">File</span>
            </button>
            <button
              onClick={() => setMode('api')}
              className={`flex flex-col items-center justify-center p-3 rounded-lg border transition-all ${
                mode === 'api' ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-background text-textDim hover:bg-surface-alt'
              }`}
            >
              <Globe size={20} className="mb-2" />
              <span className="text-[10px] font-bold uppercase">API</span>
            </button>
            <button
              onClick={() => setMode('db')}
              className={`flex flex-col items-center justify-center p-3 rounded-lg border transition-all ${
                mode === 'db' ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-background text-textDim hover:bg-surface-alt'
              }`}
            >
              <Database size={20} className="mb-2" />
              <span className="text-[10px] font-bold uppercase">DB</span>
            </button>
          </div>
        </div>

        {mode === 'file' && (
          <div>
            <label htmlFor="datasource-format" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Formato File</label>
            <div className="flex gap-2">
              {[
                { id: 'csv', label: 'CSV', icon: FileText },
                { id: 'json', label: 'JSON', icon: Code },
                { id: 'xml', label: 'XML', icon: FileText },
              ].map(({ id, label, icon: Icon }) => (
                <button
                  key={id}
                  onClick={() => handleFileFormatChange(id as 'csv' | 'json' | 'xml')}
                  className={`flex-1 flex items-center justify-center gap-2 p-2 rounded-lg border text-xs transition-all ${
                    formData.sourceType === id ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-background text-textDim'
                  }`}
                >
                  <Icon size={14} />
                  {label}
                </button>
              ))}
            </div>
          </div>
        )}

        {mode === 'api' && (
          <div>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">URL Endpoint</label>
            <div className="relative">
              <div className="absolute left-3 top-1/2 -translate-y-1/2 text-textDim">
                <Link size={14} />
              </div>
              <input
                value={JSON.parse(formData.configJson || '{}').url || ''}
                onChange={(e) => {
                  const config = JSON.parse(formData.configJson || '{}')
                  setFormData({ ...formData, configJson: JSON.stringify({ ...config, url: e.target.value }, null, 2) })
                }}
                className="w-full p-3 pl-10 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
                placeholder="https://api.example.com/v1/data"
              />
            </div>
          </div>
        )}

        {mode === 'db' && (
          <div>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Connetti Database</label>
            <input
              value={JSON.parse(formData.configJson || '{}').connectionString || ''}
              onChange={(e) => {
                const config = JSON.parse(formData.configJson || '{}')
                setFormData({ ...formData, configJson: JSON.stringify({ ...config, connectionString: e.target.value }, null, 2) })
              }}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
              placeholder="postgresql://user:pass@localhost:5432/dbname"
            />
          </div>
        )}

        <div>
          <label htmlFor="datasource-config" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Configurazione Avanzata (JSON)</label>
          <textarea
            id="datasource-config"
            value={formData.configJson}
            onChange={(e) => setFormData({ ...formData, configJson: e.target.value })}
            rows={6}
            className={`w-full p-3 bg-background rounded-lg border text-xs font-mono resize-none focus:outline-none focus:border-primary/50 transition-colors ${
              errors.configJson ? 'border-danger bg-danger/5' : 'border-border'
            }`}
          />
          {errors.configJson && <p className="text-danger text-[10px] mt-1">{errors.configJson}</p>}
        </div>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          onClick={onCancel}
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
