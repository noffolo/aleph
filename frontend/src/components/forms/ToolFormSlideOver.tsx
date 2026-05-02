import { useState, type FormEvent } from 'react'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useToolActions } from '../../hooks/domain/useToolActions'
import type { Tool } from '../../store/types'
import { t } from '../../i18n'
import { ToolSchema } from '../../schemas'

type FormErrors = Partial<Record<string, string>>

interface ToolFormSlideOverProps {
  tool?: Tool
  title?: string
}

export function ToolFormSlideOver({ tool, title }: ToolFormSlideOverProps) {
  const { loadProjectData } = useAppActions()
  const { onCreateTool, onUpdateTool } = useToolActions(loadProjectData)
  const isEdit = tool && tool.id
  const [name, setName] = useState(tool?.name || '')
  const [description, setDescription] = useState(tool?.description || '')
  const [code, setCode] = useState(tool?.code || '')
  const [errors, setErrors] = useState<FormErrors>({})

  const [isSaving, setIsSaving] = useState(false)

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setErrors({})
    setIsSaving(true)

    const parsed = ToolSchema.safeParse({ id: tool?.id || '', name, description, code })
    if (!parsed.success) {
      setErrors(parsed.error.flatten().fieldErrors as unknown as FormErrors)
      setIsSaving(false)
      return
    }

    try {
      if (isEdit && tool?.id) {
        await onUpdateTool({ ...tool, name, description, code })
      } else {
        await onCreateTool(name, description, code)
      }
      useStore.getState().setSlideOverContent(null)
    } catch (e) {
      setErrors({ submit: 'Errore durante il salvataggio. Riprova.' })
    } finally {
      setIsSaving(false)
    }
  }

  const errorId = (field: string) => `so-tool-${field}-error`

  return (
    <form onSubmit={handleSubmit} noValidate className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? t('tools.edit') : t('tools.create'))}</h3>

      {errors.submit && (
        <p role="alert" className="text-danger text-sm bg-danger/10 border border-danger/30 rounded-lg px-3 py-2">
          {errors.submit}
        </p>
      )}

      <div className="space-y-3">
        <div>
          <label htmlFor="so-tool-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            id="so-tool-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            minLength={2}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder={t('tools.form.name')}
            aria-describedby={errors.name ? errorId('name') : undefined}
            aria-invalid={errors.name ? true : undefined}
          />
          {errors.name && <p id={errorId('name')} role="alert" className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>

        <div>
          <label htmlFor="so-tool-description" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            id="so-tool-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={t('tools.form.description')}
          />
        </div>

        <div>
          <label htmlFor="so-tool-code" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Codice</label>
          <textarea
            id="so-tool-code"
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
           type="button"
           onClick={() => useStore.getState().setSlideOverContent(null)}
           className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
         >
          {t('confirmDialog.cancel')}
        </button>
        <button
          type="submit"
          disabled={isSaving}
          className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors disabled:opacity-50 flex items-center justify-center space-x-2"
        >
          {isSaving && <div className="w-3 h-3 border-2 border-background border-t-transparent rounded-full animate-spin" />}
          <span>{isSaving ? t('generic.saving') : (isEdit ? t('tools.edit') : t('tools.create'))}</span>
        </button>
      </div>
    </form>
  )
}
