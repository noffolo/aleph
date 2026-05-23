import { useState, type FormEvent } from 'react'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useDataSourceActions } from '../../hooks/domain/useDataSourceActions'
import { Upload, Globe, Database, FileText, Code, Link, ChevronLeft, ChevronRight, Check } from 'lucide-react'
import { t } from '../../i18n'

interface DataSourceFormSlideOverProps {
  title?: string
}

type Mode = 'file' | 'api' | 'db'

export function DataSourceFormSlideOver({ title }: DataSourceFormSlideOverProps) {
  const { loadProjectData } = useAppActions()
  const { onAddSource } = useDataSourceActions(loadProjectData)

  const [step, setStep] = useState(1)
  const [mode, setMode] = useState<Mode>('file')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [sourceType, setSourceType] = useState('csv')
  const [configJson, setConfigJson] = useState('{}')
  const [errors, setErrors] = useState<Partial<Record<string, string>>>({})

  const updateConfig = (key: string, value: string) => {
    try {
      const config = JSON.parse(configJson || '{}')
      config[key] = value
      setConfigJson(JSON.stringify(config, null, 2))
    } catch {
      setConfigJson(`{\n  "${key}": "${value}"\n}`)
    }
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const content = event.target?.result as string
      const config = {
        path: file.name,
        content: content,
        size: file.size,
        type: file.type || 'text/plain'
      }
      setConfigJson(JSON.stringify(config, null, 2))
    }
    reader.readAsText(file)
  }

  const validateStep = (): boolean => {
    const newErrors: Partial<Record<string, string>> = {}

      if (step === 1) {
        if (!name.trim()) newErrors.name = 'Il nome è obbligatorio'
      } else if (step === 3) {
        try {
          JSON.parse(configJson)
        } catch {
          newErrors.configJson = 'Il JSON di configurazione non è valido'
        }

        const config = JSON.parse(configJson || '{}')
        if (mode === 'api' && config.url && !/^https?:\/\/.+/.test(config.url)) {
          newErrors.url = 'L\'URL deve iniziare con http o https'
        }
        if (mode === 'db' && (!config.connectionString || !config.connectionString.trim())) {
          newErrors.connectionString = 'La stringa di connessione è obbligatoria'
        }
      }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!validateStep()) return

    onAddSource({ name, sourceType, configJson })
    useStore.getState().setSlideOverContent(null)
  }

  const nextStep = () => {
    if (validateStep()) setStep(s => s + 1)
  }

  const prevStep = () => setStep(s => s - 1)

  const errorId = (field: string) => `so-ds-${field}-error`

  return (
    <form onSubmit={handleSubmit} noValidate className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <h3 className="text-xl font-bold">{title || 'Nuova Sorgente Dati'}</h3>
        <div className="flex gap-1">
          {[1, 2, 3].map(s => (
            <div
              key={s}
              className={`w-2 h-2 rounded-full transition-colors ${step === s ? 'bg-primary' : 'bg-border'}`}
            />
          ))}
        </div>
      </div>

      <div className="space-y-4">
        {step === 1 && (
          <div className="space-y-4 animate-in fade-in slide-in-from-right-4 duration-200">
            <div>
              <label htmlFor="so-ds-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome Sorgente</label>
              <input
                id="so-ds-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
                  errors.name ? 'border-danger bg-danger/5' : 'border-border'
                }`}
                placeholder={t('datasources.form.name')}
                aria-describedby={errors.name ? errorId('name') : undefined}
                aria-invalid={errors.name ? true : undefined}
              />
              {errors.name && <p id={errorId('name')} role="alert" className="text-danger text-[10px] mt-1">{errors.name}</p>}
            </div>

            <div>
              <label htmlFor="so-ds-description" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione (Opzionale)</label>
              <textarea
                id="so-ds-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
                className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50 resize-none"
                placeholder={t('datasources.form.description')}
              />
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-4 animate-in fade-in slide-in-from-right-4 duration-200">
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Scegli Tipo Sorgente</label>
            <div className="grid grid-cols-3 gap-3">
              <button
                type="button"
                onClick={() => { setMode('file'); setSourceType('csv'); }}
                className={`flex flex-col items-center justify-center p-4 rounded-lg border transition-all ${
                  mode === 'file' ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background text-textDim hover:bg-surface-alt'
                }`}
              >
                <Upload size={24} className="mb-2" />
                <span className="text-[10px] font-bold uppercase">File</span>
              </button>
              <button
                type="button"
                onClick={() => { setMode('api'); setSourceType('api'); }}
                className={`flex flex-col items-center justify-center p-4 rounded-lg border transition-all ${
                  mode === 'api' ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background text-textDim hover:bg-surface-alt'
                }`}
              >
                <Globe size={24} className="mb-2" />
                <span className="text-[10px] font-bold uppercase">API</span>
              </button>
              <button
                type="button"
                onClick={() => { setMode('db'); setSourceType('database'); }}
                className={`flex flex-col items-center justify-center p-4 rounded-lg border transition-all ${
                  mode === 'db' ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background text-textDim hover:bg-surface-alt'
                }`}
              >
                <Database size={24} className="mb-2" />
                <span className="text-[10px] font-bold uppercase">DB</span>
              </button>
            </div>

            {mode === 'file' && (
              <div className="pt-2">
                <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-2 block">Formato File</label>
                <div className="flex gap-2">
                  {[
                    { id: 'csv', label: 'CSV', icon: FileText },
                    { id: 'json', label: 'JSON', icon: Code },
                    { id: 'xml', label: 'XML', icon: FileText },
                  ].map(({ id, label, icon: Icon }) => (
                    <button
                      type="button"
                      key={id}
                      onClick={() => setSourceType(id)}
                      className={`flex-1 flex items-center justify-center gap-2 p-2 rounded-lg border text-xs transition-all ${
                        sourceType === id ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background text-textDim'
                      }`}
                    >
                      <Icon size={14} />
                      {label}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4 animate-in fade-in slide-in-from-right-4 duration-200">
            {mode === 'file' && (
              <div>
                <label htmlFor="so-ds-file" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Carica File</label>
                <div className="relative">
                  <input
                    id="so-ds-file"
                    type="file"
                    accept=".csv,.json,.xml"
                    onChange={handleFileUpload}
                    className="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
                  />
                  <div className="w-full p-6 border-2 border-dashed border-border rounded-lg flex flex-col items-center justify-center text-textDim hover:border-primary/50 transition-colors bg-background">
                    <Upload size={24} className="mb-2" />
                    <span className="text-xs font-medium">Trascina qui o clicca per caricare</span>
                    <span className="text-[10px] opacity-50 mt-1">CSV, JSON, XML</span>
                  </div>
                </div>
              </div>
            )}

            {mode === 'api' && (
              <div>
                <label htmlFor="so-ds-api-url" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">URL Endpoint</label>
                <div className="relative">
                  <div className="absolute left-3 top-1/2 -translate-y-1/2 text-textDim">
                    <Link size={14} />
                  </div>
                  <input
                    id="so-ds-api-url"
                    value={JSON.parse(configJson || '{}').url || ''}
                    onChange={(e) => updateConfig('url', e.target.value)}
                    pattern="^https?://.+"
                    className={`w-full p-3 pl-10 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
                      errors.url ? 'border-danger bg-danger/5' : 'border-border'
                    }`}
                    placeholder="https://api.example.com/v1/data"
                    aria-describedby={errors.url ? errorId('url') : undefined}
                    aria-invalid={errors.url ? true : undefined}
                  />
                  {errors.url && <p id={errorId('url')} role="alert" className="text-danger text-[10px] mt-1">{errors.url}</p>}
                </div>
              </div>
            )}

            {mode === 'db' && (
              <div>
                <label htmlFor="so-ds-conn-string" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">ConnectionString</label>
                <input
                  id="so-ds-conn-string"
                  value={JSON.parse(configJson || '{}').connectionString || ''}
                  onChange={(e) => updateConfig('connectionString', e.target.value)}
                  className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
                    errors.connectionString ? 'border-danger bg-danger/5' : 'border-border'
                  }`}
                  placeholder="postgresql://user:pass@localhost:5432/dbname"
                  aria-describedby={errors.connectionString ? errorId('conn-string') : undefined}
                  aria-invalid={errors.connectionString ? true : undefined}
                />
                {errors.connectionString && <p id={errorId('conn-string')} role="alert" className="text-danger text-[10px] mt-1">{errors.connectionString}</p>}
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label htmlFor="so-ds-start-date" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Data Inizio (opzionale)</label>
                <input
                  id="so-ds-start-date"
                  type="date"
                  value={(() => { try { return JSON.parse(configJson).start_date || '' } catch { return '' } })()}
                  onChange={(e) => {
                    const config = JSON.parse(configJson || '{}')
                    if (e.target.value) {
                      config.start_date = e.target.value
                    } else {
                      delete config.start_date
                    }
                    setConfigJson(JSON.stringify(config, null, 2))
                  }}
                  className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50 transition-colors"
                />
              </div>
              <div>
                <label htmlFor="so-ds-end-date" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Data Fine (opzionale)</label>
                <input
                  id="so-ds-end-date"
                  type="date"
                  value={(() => { try { return JSON.parse(configJson).end_date || '' } catch { return '' } })()}
                  onChange={(e) => {
                    const config = JSON.parse(configJson || '{}')
                    if (e.target.value) {
                      config.end_date = e.target.value
                    } else {
                      delete config.end_date
                    }
                    setConfigJson(JSON.stringify(config, null, 2))
                  }}
                  className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50 transition-colors"
                />
              </div>
            </div>

            <div>
            <label htmlFor="so-ds-advanced-config" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Configurazione Avanzata (JSON)</label>
            <textarea
              id="so-ds-advanced-config"
              value={configJson}
                onChange={(e) => setConfigJson(e.target.value)}
                rows={6}
                className={`w-full p-3 bg-background rounded-lg border text-xs font-mono resize-none focus:outline-none focus:border-primary/50 transition-colors ${
                  errors.configJson ? 'border-danger bg-danger/5' : 'border-border'
                }`}
                aria-describedby={errors.configJson ? errorId('config') : undefined}
                aria-invalid={errors.configJson ? true : undefined}
              />
              {errors.configJson && <p id={errorId('config')} role="alert" className="text-danger text-[10px] mt-1">{errors.configJson}</p>}
            </div>
          </div>
        )}
      </div>

      <div className="flex gap-3 pt-4">
         <button
           type="button"
           onClick={step === 1 ? () => useStore.getState().setSlideOverContent(null) : prevStep}
           className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border flex items-center justify-center gap-2"
         >
          {step === 1 ? t('confirmDialog.cancel') : <><ChevronLeft size={16} /> Indietro</>}
        </button>

        {step < 3 ? (
          <button
            type="button"
            onClick={nextStep}
            className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors flex items-center justify-center gap-2"
          >
            Avanti <ChevronRight size={16} />
          </button>
        ) : (
          <button
            type="submit"
            className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors flex items-center justify-center gap-2"
          >
            <Check size={16} /> Crea Sorgente
          </button>
        )}
      </div>
    </form>
  )
}
