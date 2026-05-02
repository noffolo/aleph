import { useState, type FormEvent } from 'react'
import { useComponentActions } from '../../hooks/domain/useComponentActions'
import { t } from '../../i18n'
import { RegistryComponentSchema } from '../../schemas'

type FormErrors = Partial<Record<string, string>>

interface ComponentFormSlideOverProps {
  title?: string
  onClose: () => void
}

export function ComponentFormSlideOver({ title, onClose }: ComponentFormSlideOverProps) {
  const { onRegisterComponent } = useComponentActions()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [type, setType] = useState('skill')
  const [category, setCategory] = useState('generative')
  const [source, setSource] = useState('user')
  const [status, setStatus] = useState('pending')
  const [approvalStatus, setApprovalStatus] = useState('pending')
  const [configSchemaJson, setConfigSchemaJson] = useState('{}')
  const [executionCommand, setExecutionCommand] = useState('')
  const [dependenciesJson, setDependenciesJson] = useState('[]')
  const [inputSchemaJson, setInputSchemaJson] = useState('{}')
  const [outputSchemaJson, setOutputSchemaJson] = useState('{}')
  const [promptTemplate, setPromptTemplate] = useState('')
  const [toolIdsJson, setToolIdsJson] = useState('[]')
  const [errors, setErrors] = useState<FormErrors>({})

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setErrors({})

    if (!name.trim()) {
      setErrors({ name: 'Il nome è obbligatorio' })
      return
    }

    try {
      JSON.parse(configSchemaJson)
      JSON.parse(dependenciesJson)
      JSON.parse(inputSchemaJson)
      JSON.parse(outputSchemaJson)
      JSON.parse(toolIdsJson)
    } catch {
      setErrors({ configSchemaJson: 'JSON non valido' })
      return
    }

    const parsed = RegistryComponentSchema.safeParse({
      id: '',
      name,
      description,
      version: '1.0.0',
      type,
      category,
      source,
      status,
      approvalStatus,
      configSchemaJson,
      executionCommand,
      dependenciesJson,
      inputSchemaJson,
      outputSchemaJson,
      promptTemplate,
      toolIdsJson,
    })
    if (!parsed.success) {
      setErrors(parsed.error.flatten().fieldErrors as FormErrors)
      return
    }
    onRegisterComponent(parsed.data as unknown as import('../../store/types').RegistryComponent)

    onClose()
  }

  const errorId = (field: string) => `so-comp-${field}-error`

  return (
    <form onSubmit={handleSubmit} noValidate className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{title || t('components.register')}</h3>

      <div className="space-y-3 max-h-[70vh] overflow-y-auto pr-2">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="so-comp-name" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
            <input
              id="so-comp-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              minLength={2}
              className={`w-full p-3 bg-background rounded-lg border text-sm focus:outline-none focus:border-primary/50 ${
                errors.name ? 'border-danger bg-danger/5' : 'border-border'
              }`}
              placeholder={t('components.form.name')}
              aria-describedby={errors.name ? errorId('name') : undefined}
              aria-invalid={errors.name ? true : undefined}
            />
            {errors.name && <p id={errorId('name')} role="alert" className="text-danger text-[10px] mt-1">{errors.name}</p>}
          </div>

          <div>
            <label htmlFor="so-comp-type" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tipo</label>
            <select
              id="so-comp-type"
              value={type}
              onChange={(e) => setType(e.target.value)}
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
          <label htmlFor="so-comp-description" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
          <textarea
            id="so-comp-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={t('components.form.description')}
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="so-comp-category" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Categoria</label>
            <select
              id="so-comp-category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
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
            <label htmlFor="so-comp-source" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Sorgente</label>
            <select
              id="so-comp-source"
              value={source}
              onChange={(e) => setSource(e.target.value)}
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
            <label htmlFor="so-comp-status" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Stato</label>
            <select
              id="so-comp-status"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
            >
              <option value="pending">In Attesa</option>
              <option value="active">Attivo</option>
              <option value="inactive">Inattivo</option>
              <option value="deprecated">Deprecato</option>
            </select>
          </div>

          <div>
            <label htmlFor="so-comp-approval" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Approvazione</label>
            <select
              id="so-comp-approval"
              value={approvalStatus}
              onChange={(e) => setApprovalStatus(e.target.value)}
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
          <label htmlFor="so-comp-configschema" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Config (JSON)</label>
          <textarea
            id="so-comp-configschema"
            value={configSchemaJson}
            onChange={(e) => setConfigSchemaJson(e.target.value)}
            rows={2}
            className={`w-full p-3 bg-background rounded-lg border text-xs font-mono resize-none focus:outline-none focus:border-primary/50 ${
              errors.configSchemaJson ? 'border-danger bg-danger/5' : 'border-border'
            }`}
            placeholder='{"fields": []}'
            aria-describedby={errors.configSchemaJson ? errorId('configschema') : undefined}
            aria-invalid={errors.configSchemaJson ? true : undefined}
          />
          {errors.configSchemaJson && <p id={errorId('configschema')} role="alert" className="text-danger text-[10px] mt-1">{errors.configSchemaJson}</p>}
        </div>

        <div>
          <label htmlFor="so-comp-command" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Comando Esecuzione</label>
          <input
            id="so-comp-command"
            value={executionCommand}
            onChange={(e) => setExecutionCommand(e.target.value)}
            className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono focus:outline-none focus:border-primary/50"
            placeholder="python run_skill.py"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="so-comp-dependencies" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Dependencies (JSON)</label>
            <textarea
              id="so-comp-dependencies"
              value={dependenciesJson}
              onChange={(e) => setDependenciesJson(e.target.value)}
              rows={3}
              className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
              placeholder='["library1", "library2"]'
            />
          </div>

          <div>
            <label htmlFor="so-comp-toolids" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tool IDs (JSON)</label>
            <textarea
              id="so-comp-toolids"
              value={toolIdsJson}
              onChange={(e) => setToolIdsJson(e.target.value)}
              rows={3}
              className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
              placeholder='["tool1", "tool2"]'
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="so-comp-inputschema" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Input (JSON)</label>
            <textarea
              id="so-comp-inputschema"
              value={inputSchemaJson}
              onChange={(e) => setInputSchemaJson(e.target.value)}
              rows={3}
              className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
              placeholder='{"parameters": []}'
            />
          </div>

          <div>
            <label htmlFor="so-comp-outputschema" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Output (JSON)</label>
            <textarea
              id="so-comp-outputschema"
              value={outputSchemaJson}
              onChange={(e) => setOutputSchemaJson(e.target.value)}
              rows={3}
              className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
              placeholder='{"result": {}}'
            />
          </div>
        </div>

        <div>
          <label htmlFor="so-comp-prompt" className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Prompt Template</label>
          <textarea
            id="so-comp-prompt"
            value={promptTemplate}
            onChange={(e) => setPromptTemplate(e.target.value)}
            rows={4}
            className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            placeholder={t('components.form.systemPrompt')}
          />
        </div>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          type="button"
          onClick={onClose}
          className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
        >
          {t('confirmDialog.cancel')}
        </button>
        <button
          type="submit"
          className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
        >
          {t('components.register')}
        </button>
      </div>
    </form>
  )
}
