import { useEffect, Suspense, useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { WorkspaceOnboarding } from './components/WorkspaceOnboarding'
import { CopilotView } from './components/CopilotView'
import { CommandPalette } from './components/CommandPalette'
import { SetupWizard } from './components/SetupWizard'
import { AlephErrorBoundary } from './components/AlephErrorBoundary'
import { StatusBar } from './components/terminal'
import { InlineRenderer } from './components/terminal/InlineRenderer'
import { SlideOverPanel } from './components/terminal/SlideOverPanel'
import { TerminalEffects } from './components/terminal/TerminalEffects'
import { useStore } from './store/useStore'
import { useAppActions } from './hooks/useAppActions'
import { useViewActions } from './hooks/useViewActions'
import { setApiKey, getStoredApiKey } from './api/client'
import { projectClient, queryClient, authClient } from './api/factory'

function SlideOverContent() {
  const store = useStore()
  const actions = useViewActions()
  const content = store.slideOverContent
  if (!content) return null

  switch (content.type) {
    case 'skill': {
      const skill = content.data as import('./store/types').Skill | undefined
      if (!skill || !skill.id) return null
      const skillId = skill.id
      return (
        <div className="p-6 space-y-4">
          <h3 className="text-xl font-bold">{skill.name || content.title}</h3>
          <p className="text-textMuted">{skill.description || 'Nessuna descrizione'}</p>
          {skill.toolIds && skill.toolIds.length > 0 && (
            <div className="mb-2">
              <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-2">Strumenti Associati</div>
              <div className="flex flex-wrap gap-2">
                {skill.toolIds.map((tid: string) => {
                  const tool = store.tools.find((t: any) => t.id === tid)
                  return <span key={tid} className="text-[10px] bg-primary/10 text-primary px-2 py-1 rounded font-mono">{tool?.name || tid}</span>
                })}
              </div>
            </div>
          )}
          <div>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Parametri Input (JSON)</label>
            <textarea
              value={store.sandboxInput}
              onChange={(e) => store.setSandboxInput(e.target.value)}
              rows={3}
              className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            />
          </div>
          <button
            onClick={() => actions.skillsActions.onRunSkill(skillId)}
            className="w-full py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
          >
            Esegui Skill nel Sandbox
          </button>
        </div>
      )
    }
    case 'tool': {
      const tool = content.data as import('./store/types').Tool | undefined
      if (!tool || !tool.id) return null
      const toolId = tool.id
      return (
        <div className="p-6 space-y-4">
          <h3 className="text-xl font-bold">{tool.name || content.title}</h3>
          <p className="text-textMuted">{tool.description || 'Nessuna descrizione'}</p>
          <div className="bg-background p-4 rounded-lg border border-border">
            <pre className="text-xs font-mono text-textMuted whitespace-pre-wrap">{tool.code || '// Nessun codice'}</pre>
          </div>
          <div>
            <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Parametri Input (JSON)</label>
            <textarea
              value={store.sandboxInput}
              onChange={(e) => store.setSandboxInput(e.target.value)}
              rows={3}
              className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
            />
          </div>
          <button
            onClick={() => actions.toolsActions.onExecuteTool(toolId)}
            className="w-full py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
          >
            Esegui Tool nel Sandbox
          </button>
        </div>
      )
    }
    case 'sandbox': {
      const result = content.data as import('./store/types').SandboxResult | undefined
      return (
        <div className="space-y-4">
          <div className={`text-sm font-bold ${result?.exitCode === 0 ? 'text-success' : 'text-danger'}`}>
            Exit Code: {result?.exitCode ?? 'N/A'}
          </div>
          {result?.stdout && (
            <div className="bg-background p-4 rounded-lg border border-border">
              <div className="text-[10px] font-bold text-textDim uppercase mb-2">Stdout</div>
              <pre className="text-xs font-mono text-textMuted whitespace-pre-wrap">{result.stdout}</pre>
            </div>
          )}
          {result?.stderr && (
            <div className="bg-danger/10 p-4 rounded-lg border border-danger/20">
              <div className="text-[10px] font-bold text-danger uppercase mb-2">Stderr</div>
              <pre className="text-xs font-mono text-danger whitespace-pre-wrap">{result.stderr}</pre>
            </div>
          )}
          {result?.metricsJson && (
            <div className="text-[10px] text-textDim font-mono">Metrics: {result.metricsJson}</div>
          )}
        </div>
      )
    }
     case 'agent-form': {
      const agent = content.data as import('./store/types').Agent | undefined
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
            skillIds: agent.skillIds || []
          })
        } else {
          actions.agentsActions.onCreateAgent(name, model, systemPrompt, provider, apiKey, baseUrl)
        }
        
        store.setSlideOverContent(null)
      }

      return (
        <div className="p-6 space-y-4">
          <h3 className="text-xl font-bold">{content.title || (isEdit ? 'Modifica Agente' : 'Nuovo Agente')}</h3>
          
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
     
     case 'skill-form': {
       const { tools } = content.data as { tools: import('./store/types').Tool[] }
       const skill = content.data as import('./store/types').Skill | undefined
       const isEdit = skill && skill.id
       const [name, setName] = useState(skill?.name || '')
       const [description, setDescription] = useState(skill?.description || '')
       const [toolIds, setToolIds] = useState<string[]>(skill?.toolIds || [])
       
       const handleSubmit = () => {
         if (!name.trim()) {
           alert('Il nome è obbligatorio')
           return
         }
         
         if (isEdit && skill?.id) {
           // TODO: Implement updateSkill when API is available
           alert('Update skill non ancora implementato')
           store.setSlideOverContent(null)
         } else {
           actions.skillsActions.onCreateSkill(name, description, toolIds)
           store.setSlideOverContent(null)
         }
       }
       
       return (
         <div className="p-6 space-y-4">
           <h3 className="text-xl font-bold">{content.title || (isEdit ? 'Modifica Skill' : 'Nuova Skill')}</h3>
           
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
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
               <textarea
                 value={description}
                 onChange={(e) => setDescription(e.target.value)}
                 rows={3}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
                 placeholder="Descrivi la capacità di questa skill..."
               />
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Strumenti Associati</label>
               <div className="grid grid-cols-2 gap-2 max-h-40 overflow-y-auto p-2 bg-background rounded-lg border border-border">
                 {tools.map(t => (
                   <label key={t.id} className="flex items-center space-x-2 p-2 hover:bg-surface-alt rounded cursor-pointer">
                     <input
                       type="checkbox"
                       checked={toolIds.includes(t.id)}
                       onChange={(e) => {
                         if (e.target.checked) {
                           setToolIds([...toolIds, t.id])
                         } else {
                           setToolIds(toolIds.filter(id => id !== t.id))
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
               onClick={() => store.setSlideOverContent(null)}
               className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
             >
               Annulla
             </button>
             <button
               onClick={handleSubmit}
               className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
             >
               {isEdit ? 'Aggiorna Skill' : 'Crea Skill'}
             </button>
           </div>
         </div>
       )
     }
     
     case 'tool-form': {
       const tool = content.data as import('./store/types').Tool | undefined
       const isEdit = tool && tool.id
       const [name, setName] = useState(tool?.name || '')
       const [description, setDescription] = useState(tool?.description || '')
       const [code, setCode] = useState(tool?.code || '')
       
       const handleSubmit = () => {
         if (!name.trim()) {
           alert('Il nome è obbligatorio')
           return
         }
         
         if (isEdit && tool?.id) {
           // TODO: Implement updateTool when API is available
           alert('Update tool non ancora implementato')
           store.setSlideOverContent(null)
         } else {
           actions.toolsActions.onCreateTool(name, description, code)
           store.setSlideOverContent(null)
         }
       }
       
       return (
         <div className="p-6 space-y-4">
           <h3 className="text-xl font-bold">{content.title || (isEdit ? 'Modifica Tool' : 'Nuovo Tool')}</h3>
           
           <div className="space-y-3">
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
               <input
                 value={name}
                 onChange={(e) => setName(e.target.value)}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
                 placeholder="Es: Analizzatore CSV"
               />
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
               <textarea
                 value={description}
                 onChange={(e) => setDescription(e.target.value)}
                 rows={2}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
                 placeholder="Descrivi cosa fa questo tool..."
               />
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Codice</label>
               <textarea
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
               onClick={() => store.setSlideOverContent(null)}
               className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
             >
               Annulla
             </button>
             <button
               onClick={handleSubmit}
               className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
             >
               {isEdit ? 'Aggiorna Tool' : 'Crea Tool'}
             </button>
           </div>
         </div>
       )
     }
     
     case 'datasource-form': {
       const [name, setName] = useState('')
       const [sourceType, setSourceType] = useState('csv')
       const [configJson, setConfigJson] = useState('{}')
       
       const handleSubmit = () => {
         if (!name.trim()) {
           alert('Il nome è obbligatorio')
           return
         }
         
         try {
           JSON.parse(configJson)
         } catch (e) {
           alert('Config JSON non valido')
           return
         }
         
         actions.dataSourcesActions.onAddSource({ name, sourceType, configJson })
         store.setSlideOverContent(null)
       }
       
       return (
         <div className="p-
6 space-y-4">
           <h3 className="text-xl font-bold">{content.title || 'Nuova Sorgente Dati'}</h3>
           
           <div className="space-y-3">
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
               <input
                 value={name}
                 onChange={(e) => setName(e.target.value)}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
                 placeholder="Es: Dati CSV clienti"
               />
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tipo Sorgente</label>
               <select
                 value={sourceType}
                 onChange={(e) => setSourceType(e.target.value)}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
               >
                 <option value="csv">CSV</option>
                 <option value="api">API REST</option>
                 <option value="database">Database</option>
                 <option value="json">JSON File</option>
                 <option value="xml">XML</option>
               </select>
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Configurazione JSON</label>
               <textarea
                 value={configJson}
                 onChange={(e) => setConfigJson(e.target.value)}
                 rows={6}
                 className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
                  placeholder={`{\n  "url": "https://...",\n  "format": "csv",\n  "columns": []\n}`}
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
               Crea Sorgente
             </button>
           </div>
         </div>
       )
     }
     
     case 'component-form': {
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
       
       const handleSubmit = () => {
         if (!name.trim()) {
           alert('Il nome è obbligatorio')
           return
         }
         
         try {
           JSON.parse(configSchemaJson)
           JSON.parse(dependenciesJson)
           JSON.parse(inputSchemaJson)
           JSON.parse(outputSchemaJson)
           JSON.parse(toolIdsJson)
         } catch (e) {
           alert('JSON non valido')
           return
         }
         
         actions.componentsActions.onRegisterComponent({
           name,
           description,
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
         
         store.setSlideOverContent(null)
       }
       
       return (
         <div className="p-6 space-y-4">
           <h3 className="text-xl font-bold">{content.title || 'Registra Componente'}</h3>
           
           <div className="space-y-3 max-h-[70vh] overflow-y-auto pr-2">
             <div className="grid grid-cols-2 gap-3">
               <div>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Nome</label>
                 <input
                   value={name}
                   onChange={(e) => setName(e.target.value)}
                   className="w-full p-3 bg-background rounded-lg border border-border text-sm focus:outline-none focus:border-primary/50"
                   placeholder="Es: Analizzatore CSV"
                 />
               </div>
               
               <div>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tipo</label>
                 <select
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
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Descrizione</label>
               <textarea
                 value={description}
                 onChange={(e) => setDescription(e.target.value)}
                 rows={2}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono resize-none focus:outline-none focus:border-primary/50"
                 placeholder="Descrivi il componente..."
               />
             </div>
             
             <div className="grid grid-cols-2 gap-3">
               <div>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Categoria</label>
                 <select
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
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Sorgente</label>
                 <select
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
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Stato</label>
                 <select
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
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Approvazione</label>
                 <select
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
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Config (JSON)</label>
               <textarea
                 value={configSchemaJson}
                 onChange={(e) => setConfigSchemaJson(e.target.value)}
                 rows={2}
                 className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
                 placeholder='{"fields": []}'
               />
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Comando Esecuzione</label>
               <input
                 value={executionCommand}
                 onChange={(e) => setExecutionCommand(e.target.value)}
                 className="w-full p-3 bg-background rounded-lg border border-border text-sm font-mono focus:outline-none focus:border-primary/50"
                 placeholder="python run_skill.py"
               />
             </div>
             
             <div className="grid grid-cols-2 gap-3">
               <div>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Dependencies (JSON)</label>
                 <textarea
                   value={dependenciesJson}
                   onChange={(e) => setDependenciesJson(e.target.value)}
                   rows={3}
                   className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
                   placeholder='["library1", "library2"]'
                 />
               </div>
               
               <div>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Tool IDs (JSON)</label>
                 <textarea
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
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Input (JSON)</label>
                 <textarea
                   value={inputSchemaJson}
                   onChange={(e) => setInputSchemaJson(e.target.value)}
                   rows={3}
                   className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
                   placeholder='{"parameters": []}'
                 />
               </div>
               
               <div>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Schema Output (JSON)</label>
                 <textarea
                   value={outputSchemaJson}
                   onChange={(e) => setOutputSchemaJson(e.target.value)}
                   rows={3}
                   className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
                   placeholder='{"result": {}}'
                 />
               </div>
             </div>
             
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Prompt Template</label>
               <textarea
                 value={promptTemplate}
                 onChange={(e) => setPromptTemplate(e.target.value)}
                 rows={4}
                 className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
                 placeholder="Tu sei un..."
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
               Registra Componente
             </button>
           </div>
         </div>
       )
     }
     
     case 'component-detail': {
       const { componentId } = content.data as { componentId: string }
       const component = store.registryComponents.find(c => c.id === componentId)
       
       if (!component) return null
       
       return (
         <div className="p-6 space-y-4">
           <h3 className="text-xl font-bold">{component.name || content.title}</h3>
           <p className="text-textMuted">{component.description || 'Nessuna descrizione'}</p>
           
           <div className="grid grid-cols-2 gap-4">
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Tipo</div>
               <div className="text-sm">{component.type}</div>
             </div>
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Categoria</div>
               <div className="text-sm">{component.category}</div>
             </div>
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Sorgente</div>
               <div className="text-sm">{component.source}</div>
             </div>
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Stato</div>
               <div className="text-sm">{component.status}</div>
             </div>
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Approvazione</div>
               <div className="text-sm">{component.approvalStatus}</div>
             </div>
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1">Versione</div>
               <div className="text-sm">{component.version}</div>
             </div>
           </div>
           
           {component.promptTemplate && (
             <div>
               <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-2">Prompt Template</div>
               <div className="p-3 bg-background rounded-lg border border-border text-xs font-mono whitespace-pre-wrap">{component.promptTemplate}</div>
             </div>
           )}
           
           <div className="flex gap-3 pt-2">
             <button
               onClick={() => store.setSlideOverContent(null)}
               className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
             >
               Chiudi
             </button>
           </div>
         </div>
       )
     }
     
     case 'asset': {
       const { assetId } = content.data as { assetId: string }
       const asset = store.assets.find(a => a.id === assetId)
       
       if (!asset) return null
       
       return (
         <div className="p-6 space-y-4">
           <h3 className="text-xl font-bold">{asset.name || content.title}</h3>
           <p className="text-textMuted">Asset Type: {asset.type}</p>
           
           <div className="space-y-4">
             <div>
               <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Preview</label>
               <div className="bg-background p-4 rounded-lg border border-border text-sm">
                 Mostra contenuto asset...
               </div>
             </div>
             
             <div className="flex gap-3">
               <button
                 onClick={() => actions.libraryActions.onGetAssetContent(assetId)}
                 className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
               >
                 Vedi Contenuto
               </button>
               <button
                 onClick={() => actions.libraryActions.onGeneratePdf(assetId)}
                 className="flex-1 py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
               >
                 Genera PDF
               </button>
             </div>
           </div>
           
           <div className="flex gap-3 pt-2">
             <button
               onClick={() => store.setSlideOverContent(null)}
               className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
             >
               Chiudi
             </button>
           </div>
         </div>
       )
     }
     
     case 'detail': {
       const detailData = content.data as Record<string, unknown>
       const title = content.title || 'Dettaglio'
       
       return (
         <div className="p-6 space-y-4">
           <h3 className="text-xl font-bold">{title}</h3>
           
           <div className="space-y-3">
             {Object.entries(detailData).map(([key, value]) => (
               <div key={key}>
                 <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">{key}</label>
                 <div className="bg-background p-3 rounded-lg border border-border text-sm font-mono">
                   {typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
                 </div>
               </div>
             ))}
           </div>
           
           <div className="flex gap-3 pt-2">
             <button
               onClick={() => store.setSlideOverContent(null)}
               className="flex-1 py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
             >
               Chiudi
             </button>
           </div>
         </div>
       )
     }
     
     default:
       return null
  }
}

function App() {
  const store = useStore()
  const actions = useAppActions()
  const { handleError, loadProjectData, onSend, onConfirmAction } = actions

  useEffect(() => {
    projectClient.listProjects({}).then((res: any) => store.setProjects(res.projects)).catch((e) => handleError(e, 'listProjects'))
  }, [])

  useEffect(() => { loadProjectData() }, [loadProjectData])

  useEffect(() => {
    if (!store.projectID || !store.selectedObject) return
    store.setIsExplorerLoading(true)
    queryClient.executeQuery({ projectId: store.projectID, objectType: store.selectedObject, limit: 100 }).then((res: any) => {
      store.setData(res)
    }).catch((e) => {
      store.setData(null)
      handleError(e, 'executeQuery')
    }).finally(() => store.setIsExplorerLoading(false))

    queryClient.getDataStats({ projectId: store.projectID, objectType: store.selectedObject }).then((res: any) => {
      store.setDataHealthStats(res.stats || [])
    }).catch((e) => {
      store.setDataHealthStats([])
      handleError(e, 'getDataStats')
    })
  }, [store.projectID, store.selectedObject])

  useEffect(() => {
    if (!store.projectID || !store.selectedAgent) return
    queryClient.getChatHistory({ projectId: store.projectID, agentId: store.selectedAgent }).then((res: any) => {
      if (res.messages?.length > 0) {
        store.setChat(res.messages.map((m: any) => ({
          role: m.role,
          content: m.content,
          toolCall: m.toolCall || '',
          requiresConfirmation: false,
          createdAt: m.createdAt || 0,
        })))
      }
    }).catch((e) => handleError(e, 'getChatHistory'))
  }, [store.projectID, store.selectedAgent])

  useEffect(() => {
    if (!store.searchQuery || !store.projectID) {
      store.setGlobalSearchResults(null)
      return
    }
    const timer = setTimeout(() => {
      const objMatch = store.availableObjects.find(o => o.toLowerCase().includes(store.searchQuery.toLowerCase()))
      if (objMatch) {
        queryClient.globalQuery({ projectId: store.projectID, objectType: objMatch, limit: 20 }).then((res: any) => {
          store.setGlobalSearchResults(res)
        }).catch(() => {
          store.setGlobalSearchResults(null)
        })
      } else {
        store.setGlobalSearchResults(null)
      }
    }, 400)
    return () => clearTimeout(timer)
  }, [store.searchQuery, store.projectID, store.availableObjects])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        useStore.getState().setIsCommandPaletteOpen(!useStore.getState().isCommandPaletteOpen)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  if (store.showWizard) return (
    <AlephErrorBoundary>
      <SetupWizard
        onCreateProject={async (n: string) => { const r = await projectClient.createProject({ id: n.toLowerCase(), name: n }); return r.project?.id ?? n.toLowerCase() }}
        onCreateApiKey={async (pid: string, l: string) => { const r = await authClient.createApiKey({ projectId: pid, label: l }); return r.key?.key ?? '' }}
        onComplete={(pid, key) => { setApiKey(key); store.setProjectContext(pid, key); store.setShowWizard(false); store.setShowOnboarding(false) }}
      />
    </AlephErrorBoundary>
  )

  if (store.showOnboarding) return (
    <AlephErrorBoundary>
      <WorkspaceOnboarding
        projects={store.projects}
        onSelectProject={(id, key) => { setApiKey(key); store.setProjectContext(id, key); store.setShowOnboarding(false) }}
        onDeleteProject={(id, key) => { setApiKey(key); projectClient.deleteProject({ id: id }).then(() => { projectClient.listProjects({}).then((res: any) => store.setProjects(res.projects)) }).catch((e) => handleError(e, 'deleteProject')) }}
        onCreateProject={() => store.setShowWizard(true)}
      />
    </AlephErrorBoundary>
  )

  return (
    <div className="flex h-screen bg-background text-text font-mono overflow-hidden">
      <AlephErrorBoundary>
        <CommandPalette
          isOpen={store.isCommandPaletteOpen}
          onClose={() => store.setIsCommandPaletteOpen(false)}
          availableObjects={store.availableObjects}
          projects={store.projects}
          onSelectProject={(id) => {
            const p = store.projects.find((x: any) => x.id === id)
            if (p) {
              store.setProjectContext(p.id, getStoredApiKey() || '')
              store.setShowOnboarding(false)
            } else {
              store.setShowOnboarding(true)
            }
          }}
          onSelectObject={(obj) => {
            store.setSelectedObject(obj)
            store.setInlineContent({ type: 'explore', title: obj || 'Explore' })
            store.setCurrentView('inline')
            store.setShowInlinePanel(true)
          }}
        />
        <Sidebar projectID={store.projectID} onShowOnboarding={() => store.setShowOnboarding(true)} />

        <div className="flex-1 flex flex-col overflow-hidden relative">
          {store.lastError && (
            <div className="mx-4 mt-4 px-4 py-2 bg-danger/10 border border-danger/30 text-danger rounded text-sm font-mono flex items-center justify-between">
              <span>{store.lastError}</span>
              <button onClick={() => store.setLastError(null)} className="text-danger/60 hover:text-danger ml-4">✕</button>
            </div>
          )}

          <main className="flex-1 overflow-hidden p-4">
             <CopilotView
               agents={store.agents}
               selectedAgent={store.selectedAgent}
               setSelectedAgent={store.setSelectedAgent}
               chat={store.chat}
               input={store.input}
               setInput={store.setInput}
               onSend={onSend}
               isStreaming={store.isStreaming}
               onCancelStream={() => store.cancelStream()}
               onConfirmAction={onConfirmAction}
               onClearChat={() => store.clearChat()}
             />
          </main>

          <AlephErrorBoundary>
            <InlineRenderer />
          </AlephErrorBoundary>

          {store.slideOverContent && (
            <AlephErrorBoundary>
              <SlideOverPanel
                isOpen={true}
                onClose={() => store.setSlideOverContent(null)}
                title={store.slideOverContent.title}
              >
                <Suspense fallback={<div className="p-4 text-textDim text-xs font-mono">Loading...</div>}>
                  <SlideOverContent />
                </Suspense>
              </SlideOverPanel>
            </AlephErrorBoundary>
          )}

          <TerminalEffects />
          <StatusBar projectID={store.projectID} ollamaHealthy={store.ollamaHealthy} nlpHealthy={store.nlpHealthy} />
        </div>
      </AlephErrorBoundary>
    </div>
  )
}

export default App