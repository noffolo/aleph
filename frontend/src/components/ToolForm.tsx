import React, { useState } from 'react'
import type { Tool } from '../store/types'
import { useStore } from '../store/useStore'
import { t } from '../i18n'

export interface ToolFormData {
  name: string
  description: string
  code: string
}

interface ToolFormProps {
  tool?: Tool | null
  onSave: (data: ToolFormData) => void
  onCancel: () => void
  title?: string
}

export function ToolForm({ tool, onSave, onCancel, title }: ToolFormProps) {
  const isEdit = !!tool?.id
  const [errors, setErrors] = useState<Partial<Record<keyof ToolFormData, string>>>({})
  const [isSaving, setIsSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  const [formData, setFormData] = useState<ToolFormData>({
    name: tool?.name || '',
    description: tool?.description || '',
    code: tool?.code || '',
  })

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof ToolFormData, string>> = {}
    if (!formData.name.trim()) {
      newErrors.name = 'Il nome è obbligatorio'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async () => {
    if (!validate()) return

    setIsSaving(true)
    setSaveError(null)
    
    try {
      const apiKey = useStore.getState().apiKey
      const method = isEdit ? 'PATCH' : 'POST'
      const url = isEdit ? `/api/v1/tools/${tool?.id}` : '/api/v1/tools'
      
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`
        },
        body: JSON.stringify({
          ...formData,
          category: 'general'
        })
      })

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}))
        throw new Error(errData.message || `Errore ${response.status}: Impossibile salvare lo strumento`)
      }

      onSave(formData)
      onCancel()
    } catch (e: any) {
      setSaveError(e.message)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? t('tools.edit') : t('tools.create'))}</h3>
      
      {saveError && (
        <div className="p-3 bg-danger/10 border border-danger/30 text-danger text-xs rounded-lg font-medium">
          {saveError}
        </div>
      )}

      <div className="space-y-3">
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={isSaving}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            } ${isSaving ? 'opacity-50 cursor-not-allowed' : ''}`}
            placeholder={t('tools.form.name')}
          />
          {errors.name && <p className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>
        
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            disabled={isSaving}
            rows={2}
            className={`w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50 ${isSaving ? 'opacity-50 cursor-not-allowed' : ''}`}
            placeholder={t('tools.form.description')}
          />
        </div>
        
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Codice</label>
          <textarea
            value={formData.code}
            onChange={(e) => setFormData({ ...formData, code: e.target.value })}
            disabled={isSaving}
            rows={8}
            className={`w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50 ${isSaving ? 'opacity-50 cursor-not-allowed' : ''}`}
            placeholder="// Implementazione del tool..."
          />
        </div>
      </div>
      
      <div className="flex gap-3 pt-2">
        <button
          onClick={onCancel}
          disabled={isSaving}
          className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border disabled:opacity-50"
        >
          {t('confirmDialog.cancel')}
        </button>
        <button
          onClick={handleSubmit}
          disabled={isSaving}
          className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors disabled:opacity-50"
        >
          {isSaving ? 'Salvataggio...' : (isEdit ? t('tools.edit') : t('tools.create'))}
        </button>
      </div>
    </div>
  )
}
