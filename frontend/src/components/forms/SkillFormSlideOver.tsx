import { useState, type FormEvent } from 'react'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useSkillActions } from '../../hooks/domain/useSkillActions'
import type { Skill, Tool } from '../../store/types'
import { t } from '../../i18n'
import { SkillSchema } from '../../schemas'

type FormErrors = Partial<Record<string, string>>

interface SkillFormSlideOverProps {
  skill?: Skill
  tools: Tool[]
  title?: string
}

export function SkillFormSlideOver({ skill, tools, title }: SkillFormSlideOverProps) {
  const { loadProjectData } = useAppActions()
  const { onCreateSkill, onUpdateSkill } = useSkillActions(loadProjectData)
  const isEdit = skill && skill.id
  const [name, setName] = useState(skill?.name || '')
  const [description, setDescription] = useState(skill?.description || '')
  const [toolIds, setToolIds] = useState<string[]>(skill?.toolIds || [])
  const [errors, setErrors] = useState<FormErrors>({})

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setErrors({})

    const parsed = SkillSchema.safeParse({ id: skill?.id || '', name, description, toolIds })
    if (!parsed.success) {
      setErrors(parsed.error.flatten().fieldErrors as unknown as FormErrors)
      return
    }

      if (isEdit && skill?.id) {
        onUpdateSkill({ ...skill, name, description, toolIds })
        useStore.getState().setSlideOverContent(null)
      } else {
        onCreateSkill(name, description, toolIds)
        useStore.getState().setSlideOverContent(null)
      }
  }

  const errorId = (field: string) => `so-skill-${field}-error`

  return (
    <form onSubmit={handleSubmit} noValidate className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? t('skills.edit') : t('skills.create'))}</h3>

      <div className="space-y-3">
        <div>
          <label htmlFor="so-skill-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            id="so-skill-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            minLength={2}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder={t('skills.form.name')}
            aria-describedby={errors.name ? errorId('name') : undefined}
            aria-invalid={errors.name ? true : undefined}
          />
          {errors.name && <p id={errorId('name')} role="alert" className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>

        <div>
          <label htmlFor="so-skill-description" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            id="so-skill-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={3}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={t('skills.form.description')}
          />
        </div>

        <div>
          <label htmlFor="so-skill-tools" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Strumenti Associati</label>
          <div id="so-skill-tools" className="grid grid-cols-2 gap-2 max-h-40 overflow-y-auto p-2 bg-background rounded-lg border border-border">
            {tools.map((t) => (
              <label key={t.id} className="flex items-center space-x-2 p-2 hover:bg-surface-alt rounded cursor-pointer">
                <input
                  type="checkbox"
                  checked={toolIds.includes(t.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setToolIds([...toolIds, t.id])
                    } else {
                      setToolIds(toolIds.filter((id) => id !== t.id))
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
           type="button"
           onClick={() => useStore.getState().setSlideOverContent(null)}
           className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
         >
          {t('confirmDialog.cancel')}
        </button>
        <button
          type="submit"
          className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
        >
          {isEdit ? t('skills.edit') : t('skills.create')}
        </button>
      </div>
    </form>
  )
}
