import React, { useState } from 'react'
import type { Skill, Tool } from '../store/types'
import { useStore } from '../store/useStore'
import { t } from '../i18n'
import { apiPost, apiPatch } from '../api/client'
import { SkillFormSchema } from '../schemas'
import type { SkillFormData } from '../schemas'

interface SkillFormProps {
  skill?: Skill | null
  tools: Tool[]
  onSave: (data: SkillFormData) => void
  onCancel: () => void
  title?: string
}

export function SkillForm({ skill, tools, onSave, onCancel, title }: SkillFormProps) {
  const isEdit = !!skill?.id
  const [errors, setErrors] = useState<Partial<Record<keyof SkillFormData, string>>>({})
  const [isSaving, setIsSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  const [formData, setFormData] = useState<SkillFormData>({
    name: skill?.name || '',
    description: skill?.description || '',
    toolIds: skill?.toolIds ?? [],
  })

  const validate = (): boolean => {
    const result = SkillFormSchema.safeParse(formData)
    if (!result.success) {
      const newErrors: Partial<Record<keyof SkillFormData, string>> = {}
      for (const issue of result.error.issues) {
        const field = issue.path[0] as keyof SkillFormData
        if (!newErrors[field]) {
          newErrors[field] = issue.message
        }
      }
      setErrors(newErrors)
      return false
    }
    setErrors({})
    return true
  }

  const handleSubmit = async () => {
    if (!validate()) return

    setIsSaving(true)
    setSaveError(null)

    try {
      const url = isEdit ? `/api/v1/skills/${skill?.id}` : '/api/v1/skills'
      
      if (isEdit) {
        await apiPatch(url, formData)
      } else {
        await apiPost(url, formData)
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
      <h3 className="text-xl font-bold">{title || (isEdit ? t('skills.edit') : t('skills.create'))}</h3>
      
      {saveError && (
        <div className="p-3 bg-danger/10 border border-danger/30 text-danger text-xs rounded-lg font-medium">
          {saveError}
        </div>
      )}

      <div className="space-y-3">
        <div>
          <label htmlFor="skill-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            id="skill-name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            disabled={isSaving}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            } ${isSaving ? 'opacity-50 cursor-not-allowed' : ''}`}
            placeholder={t('skills.form.name')}
          />
          {errors.name && <p className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>
        
        <div>
          <label htmlFor="skill-description" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            id="skill-description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            disabled={isSaving}
            rows={3}
            className={`w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50 ${isSaving ? 'opacity-50 cursor-not-allowed' : ''}`}
            placeholder={t('skills.form.description')}
          />
        </div>
        
        <div>
          <label htmlFor="skill-tools" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Strumenti Associati</label>
          <div className={`grid grid-cols-2 gap-2 max-h-40 overflow-y-auto p-2 bg-background rounded-lg border border-border ${isSaving ? 'opacity-50 cursor-not-allowed' : ''}`}>
            {tools.map(t => (
              <label key={t.id} className="flex items-center space-x-2 p-2 hover:bg-surface-alt rounded cursor-pointer">
                <input
                  type="checkbox"
                  disabled={isSaving}
                  checked={(formData.toolIds ?? []).includes(t.id)}
                  onChange={(e) => {
                    const current = formData.toolIds ?? []
                    if (e.target.checked) {
                      setFormData({ ...formData, toolIds: [...current, t.id] })
                    } else {
                      setFormData({ ...formData, toolIds: current.filter(id => id !== t.id) })
                    }
                  }}
                  className="w-4 h-4 rounded border-border focus:ring-primary"
                />
                <span className="text-sm">{t.name}</span>
              </label>
            ))}
          </div>
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
          {isSaving ? 'Salvataggio...' : (isEdit ? 'Aggiorna Skill' : t('skills.create'))}
        </button>
      </div>
    </div>
  )
}
