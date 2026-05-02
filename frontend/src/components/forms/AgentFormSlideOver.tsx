import { useState, type FormEvent } from 'react'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useAgentActions } from '../../hooks/domain/useAgentActions'
import type { Agent } from '../../store/types'
import { t } from '../../i18n'
import { AgentSchema } from '../../schemas'

type FormErrors = Partial<Record<string, string>>

interface AgentFormSlideOverProps {
  agent?: Agent
  title?: string
}

export function AgentFormSlideOver({ agent, title }: AgentFormSlideOverProps) {
  const { loadProjectData } = useAppActions()
  const { onCreateAgent, onUpdateAgent } = useAgentActions(loadProjectData)
  const isEdit = agent && agent.id
  const [name, setName] = useState(agent?.name || '')
  const [model, setModel] = useState(agent?.model || 'gpt-4o-mini')
  const [provider, setProvider] = useState(agent?.provider || 'openai')
  const [apiKey, setApiKey] = useState(agent?.apiKey || '')
  const [baseUrl, setBaseUrl] = useState(agent?.baseUrl || '')
  const [systemPrompt, setSystemPrompt] = useState(agent?.systemPrompt || '')
  const [errors, setErrors] = useState<FormErrors>({})
  const [isSaving, setIsSaving] = useState(false)

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setErrors({})
    setIsSaving(true)

    if (!name.trim()) {
      setErrors({ name: 'Il nome è obbligatorio' })
      setIsSaving(false)
      return
    }

    try {
      if (isEdit && agent?.id) {
        const parsed = AgentSchema.safeParse({
          id: agent.id,
          name,
          model,
          provider,
          apiKey,
          baseUrl,
          systemPrompt,
          skillIds: agent.skillIds || [],
        })
        if (!parsed.success) {
          setErrors(parsed.error.flatten().fieldErrors as unknown as FormErrors)
          setIsSaving(false)
          return
        }
        await onUpdateAgent(parsed.data as unknown as Agent)
      } else {
        await onCreateAgent(name, model, systemPrompt, provider, apiKey, baseUrl)
      }

      useStore.getState().setSlideOverContent(null)
    } catch (e) {
      setErrors({ submit: 'Errore durante il salvataggio. Riprova.' })
    } finally {
      setIsSaving(false)
    }
  }

  const errorId = (field: string) => `so-agent-${field}-error`

  return (
    <form onSubmit={handleSubmit} noValidate className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? t('agents.edit') : t('agents.create'))}</h3>

      {errors.submit && (
        <p role="alert" className="text-danger text-sm bg-danger/10 border border-danger/30 rounded-lg px-3 py-2">
          {errors.submit}
        </p>
      )}

      <div className="space-y-3">
        <div>
          <label htmlFor="so-agent-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            id="so-agent-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder={t('agents.form.name')}
            aria-describedby={errors.name ? errorId('name') : undefined}
            aria-invalid={errors.name ? true : undefined}
          />
          {errors.name && <p id={errorId('name')} role="alert" className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="so-agent-provider" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Provider</label>
            <select
              id="so-agent-provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="ollama">Ollama</option>
              <option value="azure">Azure OpenAI</option>
            </select>
          </div>

          <div>
            <label htmlFor="so-agent-model" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Modello</label>
            <input
              id="so-agent-model"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              required
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
              placeholder={t('agents.form.model')}
            />
          </div>
        </div>

        <div>
          <label htmlFor="so-agent-apikey" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">API Key (opzionale)</label>
          <input
            id="so-agent-apikey"
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder={t('agents.form.apiKey')}
          />
        </div>

        <div>
          <label htmlFor="so-agent-baseurl" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Base URL (opzionale)</label>
          <input
            id="so-agent-baseurl"
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder={t('agents.form.baseUrl')}
          />
        </div>

        <div>
          <label htmlFor="so-agent-prompt" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Prompt di Sistema</label>
          <textarea
            id="so-agent-prompt"
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            rows={4}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={t('agents.form.systemPrompt')}
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
          <span>{isSaving ? t('generic.saving') : (isEdit ? t('agents.edit') : t('agents.create'))}</span>
        </button>
      </div>
    </form>
  )
}
