import { useStore } from '../../store/useStore'
import type { Skill, Tool, SandboxResult, Agent, RegistryComponent } from '../../store/types'
import { AgentFormSlideOver } from '../forms/AgentFormSlideOver'
import { SkillFormSlideOver } from '../forms/SkillFormSlideOver'
import { ToolFormSlideOver } from '../forms/ToolFormSlideOver'
import { DataSourceFormSlideOver } from '../forms/DataSourceFormSlideOver'
import { SkillExecuteSlideOver } from '../forms/SkillExecuteSlideOver'
import { ToolExecuteSlideOver } from '../forms/ToolExecuteSlideOver'
import { SandboxResultSlideOver } from '../forms/SandboxResultSlideOver'
import { ComponentFormSlideOver } from '../forms/ComponentFormSlideOver'
import { ComponentDetailSlideOver } from '../forms/ComponentDetailSlideOver'
import { AssetDetailSlideOver } from '../forms/AssetDetailSlideOver'
import { DetailSlideOver } from '../forms/DetailSlideOver'
import { ScenarioComparisonView } from '../../views/ScenarioComparisonView'

export function SlideOverContent() {
  const store = useStore()
  const content = store.slideOverContent
  if (!content) return null

  switch (content.type) {
    case 'skill': {
      const skill = content.data as Skill | undefined
      if (!skill || !skill.id) return null
      return <SkillExecuteSlideOver skill={skill} title={content.title} />
    }
    case 'tool': {
      const tool = content.data as Tool | undefined
      if (!tool || !tool.id) return null
      return <ToolExecuteSlideOver tool={tool} title={content.title} />
    }
    case 'sandbox': {
      const result = content.data as SandboxResult | undefined
      return <SandboxResultSlideOver result={result} />
    }
     case 'agent-form': {
       const agent = content.data as Agent | undefined
       return <AgentFormSlideOver agent={agent} title={content.title} />
     }
     
     case 'skill-form': {
        const { tools } = content.data as { tools: Tool[] }
        const skill = content.data as Skill | undefined
        return <SkillFormSlideOver skill={skill} tools={tools} title={content.title} />
     }
     
     case 'tool-form': {
        const tool = content.data as Tool | undefined
        return <ToolFormSlideOver tool={tool} title={content.title} />
     }
     
     case 'datasource-form': {
        return <DataSourceFormSlideOver title={content.title} />
     }
     
      case 'component-form': {
        return <ComponentFormSlideOver title={content.title} onClose={() => store.setSlideOverContent(null)} />
      }
     
      case 'component-detail': {
        const { componentId } = content.data as { componentId: string }
        return <ComponentDetailSlideOver componentId={componentId} title={content.title} onClose={() => store.setSlideOverContent(null)} />
      }
      
      case 'asset': {
        const { assetId } = content.data as { assetId: string }
        return <AssetDetailSlideOver assetId={assetId} title={content.title} onClose={() => store.setSlideOverContent(null)} />
      }
      
      case 'detail': {
        const detailData = content.data as Record<string, unknown>
        return <DetailSlideOver data={detailData} title={content.title} onClose={() => store.setSlideOverContent(null)} />
      }
     
      case 'scenario-comparison': {
        return <ScenarioComparisonView />
      }
     
      default:
       return null
  }
}
