import React, { useState } from 'react'
import type { Agent } from '../store/types'
import { Eye, EyeOff } from 'lucide-react'
import { t } from '../i18n'

export interface AgentFormData {
  name: string
  provider: string
  model: string
  apiKey: string
  baseUrl: string
  systemPrompt: string
}

interface AgentFormProps {
  agent?: Agent | null
  onSave: (data: AgentFormData) => void
  onCancel: () => void
  title?: string
}

export function AgentForm({ agent, onSave, onCancel, title }: AgentFormProps) {
  const isEdit = !!agent?.id
  const [showApiKey, setShowApiKey] = useState(false)
  const [errors, setErrors] = useState<Partial<Record<keyof AgentFormData, string>>>({})

  const [formData, setFormData] = useState<AgentFormData>({
    name: agent?.name || '',
    provider: agent?.provider || 'openai',
    model: agent?.model || 'gpt-4o-mini',
    apiKey: agent?.apiKey || '',
    baseUrl: agent?.baseUrl || '',
    systemPrompt: agent?.systemPrompt || '',
  })

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof AgentFormData, string>> = {}
    
    if (!formData.name.trim()) {
      newErrors.name = 'Il nome è obbligatorio'
    }
    
    if (!formData.model.trim()) {
      newErrors.model = 'Il modello è obbligatorio'
    }

    if (formData.baseUrl && !/^https?:\/\/.+/.test(formData.baseUrl)) {
      newErrors.baseUrl = 'URL non valido (deve iniziare con http/https)'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = () => {
    if (validate()) {
      onSave(formData)
    }
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? t('agents.edit') : t('agents.create'))}</h3>
      
      <div className="space-y-3">
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
          <input
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
              errors.name ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder={t('agents.form.name')}
          />
          {errors.name && <p className="text-danger text-[10px] mt-1">{errors.name}</p>}
        </div>
        
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Provider</label>
            <select
              value={formData.provider}
              onChange={(e) => setFormData({ ...formData, provider: e.target.value })}
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
              value={formData.model}
              onChange={(e) => setFormData({ ...formData, model: e.target.value })}
              className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
                errors.model ? 'border-danger bg-danger/5' : 'border-border'
              }`}
              placeholder={t('agents.form.model')}
            />
            {errors.model && <p className="text-danger text-[10px] mt-1">{errors.model}</p>}
          </div>
        </div>
        
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">API Key (opzionale)</label>
          <div className="relative">
            <input
              type={showApiKey ? 'text' : 'password'}
              value={formData.apiKey}
              onChange={(e) => setFormData({ ...formData, apiKey: e.target.value })}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
              placeholder={t('agents.form.apiKey')}
            />
            <button
              type="button"
              onClick={() => setShowApiKey(!showApiKey)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-textDim hover:text-text transition-colors"
            >
              {showApiKey ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
          </div>
        </div>
        
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Base URL (opzionale)</label>
          <input
            value={formData.baseUrl}
            onChange={(e) => setFormData({ ...formData, baseUrl: e.target.value })}
            className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
              errors.baseUrl ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder={t('agents.form.baseUrl')}
          />
          {errors.baseUrl && <p className="text-danger text-[10px] mt-1">{errors.baseUrl}</p>}
        </div>
        
        <div>
          <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Prompt di Sistema</label>
          <textarea
            value={formData.systemPrompt}
            onChange={(e) => setFormData({ ...formData, systemPrompt: e.target.value })}
            rows={4}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={t('agents.form.systemPrompt')}
          />
        </div>
      </div>
      
      <div className="flex gap-3 pt-2">
        <button
          onClick={onCancel}
          className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
        >
          {t('confirmDialog.cancel')}
        </button>
        <button
          onClick={handleSubmit}
          className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
        >
          {isEdit ? 'Aggiorna Agente' : t('agents.create')}
        </button>
      </div>
    </div>
  )
}
