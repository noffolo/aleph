import { useState } from 'react'
import { useStore } from '../../store/useStore'
import { useViewActions } from '../../hooks/useViewActions'
import type { Agent } from '../../store/types'

interface AgentFormSlideOverProps {
  agent?: Agent
  title?: string
}

export function AgentFormSlideOver({ agent, title }: AgentFormSlideOverProps) {
  const store = useStore()
  const actions = useViewActions()
  const isEdit = agent && agent.id
  const [name, setName] = useState(agent?.name || '')
  const [model, setModel] = useState(agent?.model || 'gpt-4o-mini')
  const [provider, setProvider] = useState(agent?.provider || 'openai')
  const [apiKey, setApiKey] = useState(agent?.apiKey || '')
  const [baseUrl, setBaseUrl] = useState(agent?.baseUrl || '')
  const [systemPrompt, setSystemPrompt] = useState(agent?.systemPrompt || '')

  const handleSubmit = () => {
    if (!name.trim()) {
      alert('Il nome è obbligatorio')
      return
    }

    if (isEdit && agent?.id) {
      actions.agentsActions.onUpdateAgent({
        id: agent.id,
        name,
        model,
        provider,
        apiKey,
        baseUrl,
        systemPrompt,
        skillIds: agent.skillIds || [],
      })
    } else {
      actions.agentsActions.onCreateAgent(name, model, systemPrompt, provider, apiKey, baseUrl)
    }

    store.setSlideOverContent(null)
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? 'Modifica Agente' : 'Nuovo Agente')}</h3>

      <div className="space-y-3">
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder="Es: Analista Finanze"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Provider</label>
            <select
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
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Modello</label>
            <input
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
              placeholder="Es: gpt-4o-mini, claude-3-5-sonnet"
            />
          </div>
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">API Key (opzionale)</label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder="Inserisci solo per override globale"
          />
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Base URL (opzionale)</label>
          <input
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            placeholder="Es: https://api.openai.com/v1"
          />
        </div>

        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Prompt di Sistema</label>
          <textarea
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            rows={4}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder="Definisci il ruolo e le restrizioni dell'agente..."
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
          {isEdit ? 'Aggiorna Agente' : 'Crea Agente'}
        </button>
      </div>
    </div>
  )
}
