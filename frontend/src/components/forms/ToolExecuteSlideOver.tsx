import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useToolActions } from '../../hooks/domain/useToolActions'
import type { Tool } from '../../store/types'

interface ToolExecuteSlideOverProps {
  tool: Tool
  title?: string
}

export function ToolExecuteSlideOver({ tool, title }: ToolExecuteSlideOverProps) {
  const sandboxInput = useStore(s => s.sandboxInput)
  const setSandboxInput = useStore(s => s.setSandboxInput)
  const { loadProjectData } = useAppActions()
  const { onExecuteTool } = useToolActions(loadProjectData)
  const toolId = tool.id

  if (!tool || !tool.id) return null

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{tool.name || title}</h3>
      <p className="text-textMuted">{tool.description || 'Nessuna descrizione'}</p>
      <div className="bg-background p-4 rounded-lg border border-border">
        <pre className="text-xs font-mono text-textMuted whitespace-pre-wrap">{tool.code || '// Nessun codice'}</pre>
      </div>
      <div>
        <label className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-1 block">Parametri Input (JSON)</label>
         <textarea
           value={sandboxInput}
           onChange={(e) => setSandboxInput(e.target.value)}
           rows={3}
           className="w-full p-3 bg-background rounded-lg border border-border text-xs font-mono resize-none focus:outline-none focus:border-primary/50"
         />
      </div>
      <button
        onClick={() => onExecuteTool(toolId)}
        className="w-full py-3 bg-surface-alt text-text rounded-lg text-sm font-bold hover:bg-border transition-colors border border-border"
      >
        Esegui Tool nel Sandbox
      </button>
    </div>
  )
}
