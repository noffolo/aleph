import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useSkillActions } from '../../hooks/domain/useSkillActions'
import type { Skill } from '../../store/types'

interface SkillExecuteSlideOverProps {
  skill: Skill
  title?: string
}

export function SkillExecuteSlideOver({ skill, title }: SkillExecuteSlideOverProps) {
  const tools = useStore(s => s.tools)
  const sandboxInput = useStore(s => s.sandboxInput)
  const setSandboxInput = useStore(s => s.setSandboxInput)
  const { loadProjectData } = useAppActions()
  const { onRunSkill } = useSkillActions(loadProjectData)
  const skillId = skill.id

  if (!skill || !skill.id) return null

  return (
    <div className="p-6 space-y-4">
      <h3 className="text-xl font-bold">{skill.name || title}</h3>
      <p className="text-textMuted">{skill.description || 'Nessuna descrizione'}</p>
      {skill.toolIds && skill.toolIds.length > 0 && (
        <div className="mb-2">
          <div className="text-[10px] font-bold text-textDim uppercase tracking-widest mb-2">Strumenti Associati</div>
          <div className="flex flex-wrap gap-2">
             {skill.toolIds.map((tid: string) => {
               const tool = tools.find((t: any) => t.id === tid)
               return <span key={tid} className="text-[10px] bg-primary/10 text-primary px-2 py-1 rounded font-mono">{tool?.name || tid}</span>
             })}
          </div>
        </div>
      )}
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
        onClick={() => onRunSkill(skillId)}
        className="w-full py-3 bg-primary text-background rounded-lg text-sm font-bold hover:bg-primary-light transition-colors"
      >
        Esegui Skill nel Sandbox
      </button>
    </div>
  )
}
