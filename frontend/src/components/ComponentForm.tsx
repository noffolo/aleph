import React, { useState } from 'react'
import type { RegistryComponent } from '../store/types'
import { ComponentFormSchema } from '../schemas'
import type { ComponentFormData } from '../schemas'

interface ComponentFormProps {
  component?: RegistryComponent | null
  onSave: (data: ComponentFormData) => void
  onCancel: () => void
  title?: string
}

export function ComponentForm({ component, onSave, onCancel, title }: ComponentFormProps) {
  const isEdit = !!component?.id
  const [errors, setErrors] = useState<Partial<Record<keyof ComponentFormData, string>>>({})

  const [formData, setFormData] = useState<ComponentFormData>({
    name: component?.name || '',
    description: component?.description || '',
    type: component?.type || 'skill',
    category: component?.category || 'generative',
    source: component?.source || 'user',
    status: component?.status || 'pending',
    approvalStatus: component?.approvalStatus || 'pending',
    configSchemaJson: component?.configSchemaJson || '{}',
    executionCommand: component?.executionCommand || '',
    dependenciesJson: component?.dependenciesJson || '[]',
    inputSchemaJson: component?.inputSchemaJson || '{}',
    outputSchemaJson: component?.outputSchemaJson || '{}',
    promptTemplate: component?.promptTemplate || '',
    toolIdsJson: component?.toolIdsJson || '[]',
  })

  const validate = (): boolean => {
    const result = ComponentFormSchema.safeParse(formData)
    if (!result.success) {
      const newErrors: Partial<Record<keyof ComponentFormData, string>> = {}
      for (const issue of result.error.issues) {
        const field = issue.path[0] as keyof ComponentFormData
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

  const handleSubmit = () => {
    if (validate()) {
      onSave(formData)
    }
  }

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || (isEdit ? 'Modifica Componente' : 'Registra Componente')}</h3>
      
      <div className="space-y-3 max-h-[70vh] overflow-y-auto pr-2">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="component-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
            <input
              id="component-name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 transition-colors ${
                errors.name ? 'border-danger bg-danger/5' : 'border-border'
              }`}
              placeholder="Es: Analizzatore CSV"
            />
            {errors.name && <p className="text-danger text-[10px] mt-1">{errors.name}</p>}
          </div>
          
          <div>
            <label htmlFor="component-type" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tipo</label>
            <select
              id="component-type"
              value={formData.type}
              onChange={(e) => setFormData({ ...formData, type: e.target.value })}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="skill">Skill</option>
              <option value="tool">Tool</option>
              <option value="agent">Agente</option>
              <option value="model">Modello</option>
              <option value="pipeline">Pipeline</option>
            </select>
          </div>
        </div>
        
        <div>
          <label htmlFor="component-description" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            id="component-description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            rows={2}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder="Descrivi il componente..."
          />
        </div>
        
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="component-category" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Categoria</label>
            <select
              id="component-category"
              value={formData.category}
              onChange={(e) => setFormData({ ...formData, category: e.target.value })}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="generative">Generativo</option>
              <option value="analytical">Analitico</option>
              <option value="transformative">Trasformativo</option>
              <option value="integration">Integrazione</option>
              <option value="orchestration">Orchestrazione</option>
            </select>
          </div>
          
          <div>
            <label htmlFor="component-source" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Sorgente</label>
            <select
              id="component-source"
              value={formData.source}
              onChange={(e) => setFormData({ ...formData, source: e.target.value })}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="user">Utente</option>
              <option value="registry">Registro</option>
              <option value="imported">Importato</option>
              <option value="generated">Generato</option>
            </select>
          </div>
        </div>
        
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="component-status" className="text-[10px] font-bold text-stextDim uppercase tracking-widest mb-1 block">Stato</label>
            <select
              id="component-status"
              value={formData.status}
              onChange={(e) => setFormData({ ...formData, status: e.target.value })}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="pending">In Attesa</option>
              <option value="active">Attivo</option>
              <option value="inactive">Inattivo</option>
              <option value="deprecated">Deprecato</option>
            </select>
          </div>
          
          <div>
            <label htmlFor="component-approval" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Approvazione</label>
            <select
              id="component-approval"
              value={formData.approvalStatus}
              onChange={(e) => setFormData({ ...formData, approvalStatus: e.target.value })}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="pending">In Attesa</option>
              <option value="approved">Approvato</option>
              <option value="rejected">Rifiutato</option>
              <option value="review">In Revisione</option>
            </select>
          </div>
        </div>
        
        <div>
          <label htmlFor="component-configschema" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Config (JSON)</label>
          <textarea
            id="component-configschema"
            value={formData.configSchemaJson}
            onChange={(e) => setFormData({ ...formData, configSchemaJson: e.target.value })}
            rows={2}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder='{"fields": []}'
          />
        </div>
        
        <div>
          <label htmlFor="component-command" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Comando Esecuzione</label>
          <input
            id="component-command"
            value={formData.executionCommand}
            onChange={(e) => setFormData({ ...formData, executionCommand: e.target.value })}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono focus:outline-none focus:border-primary/50"
            placeholder="python run_skill.py"
          />
        </div>
        
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="component-dependencies" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Dependencies (JSON)</label>
            <textarea
            id="component-dependencies"
            value={formData.dependenciesJson}
            onChange={(e) => setFormData({ ...formData, dependenciesJson: e.target.value })}
            rows={3}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder='["library1", "library2"]'
          />
          </div>
          
          <div>
            <label htmlFor="component-toolids" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tool IDs (JSON)</label>
            <textarea
            id="component-toolids"
            value={formData.toolIdsJson}
            onChange={(e) => setFormData({ ...formData, toolIdsJson: e.target.value })}
            rows={3}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder='["tool1", "tool2"]'
          />
          </div>
        </div>
        
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="component-inputschema" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Input (JSON)</label>
            <textarea
            id="component-inputschema"
            value={formData.inputSchemaJson}
            onChange={(e) => setFormData({ ...formData, inputSchemaJson: e.target.value })}
            rows={3}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder='{"parameters": []}'
          />
          </div>
          
          <div>
            <label htmlFor="component-outputschema" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Output (JSON)</label>
            <textarea
            id="component-outputschema"
            value={formData.outputSchemaJson}
            onChange={(e) => setFormData({ ...formData, outputSchemaJson: e.target.value })}
            rows={3}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder='{"result": {}}'
          />
          </div>
        </div>
        
        <div>
          <label htmlFor="component-prompt" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Prompt Template</label>
          <textarea
            id="component-prompt"
            value={formData.promptTemplate}
            onChange={(e) => setFormData({ ...formData, promptTemplate: e.target.value })}
            rows={4}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder="Tu sei un..."
          />
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
          Registra Componente
        </button>
      </div>
    </div>
  )
}
