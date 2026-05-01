import React, { Suspense } from 'react'
import { AlephErrorBoundary } from '../AlephErrorBoundary'
import { t } from '../../i18n'
import { useStore } from '../../store/useStore'
import type { Skill, Tool, SandboxResult, Agent, RegistryComponent } from '../../store/types'
import { useAppActions } from '../../hooks/useAppActions'
import { useAgentActions } from '../../hooks/domain/useAgentActions'
import { useOntologyActions } from '../../hooks/domain/useOntologyActions'
import { useDataSourceActions } from '../../hooks/domain/useDataSourceActions'
import { useSkillActions } from '../../hooks/domain/useSkillActions'
import { useToolActions } from '../../hooks/domain/useToolActions'
import { useComponentActions } from '../../hooks/domain/useComponentActions'
import { useSettingsActions } from '../../hooks/domain/useSettingsActions'
import { useLibraryActions } from '../../hooks/domain/useLibraryActions'
import type { ComponentsViewProps } from '../ComponentsView'
import type { SkillsViewProps } from '../SkillsView'
import type { ToolsViewProps } from '../ToolsView'
import type { AgentsViewProps } from '../AgentsView'
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

const ExplorerView = React.lazy(() => import('../ExplorerView').then(m => ({ default: m.ExplorerView })))
const DataSourcesView = React.lazy(() => import('../DataSourcesView').then(m => ({ default: m.DataSourcesView })))
const OntologyView = React.lazy(() => import('../OntologyView').then(m => ({ default: m.OntologyView })))
const DataHealthView = React.lazy(() => import('../DataHealthView').then(m => ({ default: m.DataHealthView })))
const SettingsView = React.lazy(() => import('../SettingsView').then(m => ({ default: m.SettingsView })))
const ComponentsView = React.lazy(() => import('../ComponentsView').then(m => ({ default: m.ComponentsView })))
const SkillsView = React.lazy(() => import('../SkillsView').then(m => ({ default: m.SkillsView })))
const ToolsView = React.lazy(() => import('../ToolsView').then(m => ({ default: m.ToolsView })))
const LibraryView = React.lazy(() => import('../LibraryView').then(m => ({ default: m.LibraryView })))
const AgentsView = React.lazy(() => import('../AgentsView').then(m => ({ default: m.AgentsView })))
const OracleView = React.lazy(() => import('../OracleView').then(m => ({ default: m.OracleView })))

export const SlideOverContent = React.memo(() => {
  const slideOverContent = useStore(s => s.slideOverContent)
  const availableObjects = useStore(s => s.availableObjects)
  const selectedObject = useStore(s => s.selectedObject)
  const setSelectedObject = useStore(s => s.setSelectedObject)
  const searchQuery = useStore(s => s.searchQuery)
  const setSearchQuery = useStore(s => s.setSearchQuery)
  const activeView = useStore(s => s.activeView)
  const setActiveView = useStore(s => s.setActiveView)
  const globalSearchResults = useStore(s => s.globalSearchResults)
  const data = useStore(s => s.data)
  const setSelectedRow = useStore(s => s.setSelectedRow)
  const isExplorerLoading = useStore(s => s.isExplorerLoading)
  const agents = useStore(s => s.agents)
  const ollamaHealthy = useStore(s => s.ollamaHealthy)
  const ollamaModels = useStore(s => s.ollamaModels)
  const ontologyRaw = useStore(s => s.ontologyRaw)
  const setOntologyRaw = useStore(s => s.setOntologyRaw)
  const ingestionTasks = useStore(s => s.ingestionTasks)
  const taskLogs = useStore(s => s.taskLogs)
  const setTaskLogs = useStore(s => s.setTaskLogs)
  const dataHealthStats = useStore(s => s.dataHealthStats)
  const skills = useStore(s => s.skills)
  const tools = useStore(s => s.tools)
  const registryComponents = useStore(s => s.registryComponents)
  const apiKeys = useStore(s => s.apiKeys)
  const notificationChannels = useStore(s => s.notificationChannels)
  const assets = useStore(s => s.assets)
  const selectedAssetContent = useStore(s => s.selectedAssetContent)
  const setSelectedAssetContent = useStore(s => s.setSelectedAssetContent)
  const selectedAssetId = useStore(s => s.selectedAssetId)
  const { loadProjectData } = useAppActions()
  const { onCreateAgent, onDeleteAgent, onUpdateAgent } = useAgentActions(loadProjectData)
  const { onEmerge, onSave } = useOntologyActions(loadProjectData)
  const { onAddSource, onRunTask, onViewLogs, onDeleteTask } = useDataSourceActions(loadProjectData)
  const { onCreateSkill, onViewSkillDetail, onDeleteSkill, onRunSkill } = useSkillActions(loadProjectData)
  const { onCreateTool, onEditTool, onDeleteTool, onExecuteTool } = useToolActions(loadProjectData)
  const { onUpdateComponentStatus, onRegisterComponent, onGetComponent } = useComponentActions()
  const { onCreateApiKey, onDeleteApiKey, onSendWebhook } = useSettingsActions()
  const { onViewAsset, onDeleteAsset, onGetAssetContent, onGeneratePdf, onUploadAsset } = useLibraryActions(loadProjectData)
  const content = slideOverContent
  if (!content) return null

  const renderContent = () => {
    switch (content.type) {
    case 'explore':
      return (
        <AlephErrorBoundary key="explore">
          <ExplorerView
            availableObjects={availableObjects}
            selectedObject={selectedObject}
            setSelectedObject={setSelectedObject}
            searchQuery={searchQuery}
            setSearchQuery={setSearchQuery}
            activeView={activeView}
            setActiveView={setActiveView}
            data={searchQuery ? globalSearchResults : data}
            onRowClick={setSelectedRow}
            isLoading={isExplorerLoading}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'agent':
      return (
        <AlephErrorBoundary key="agent">
          <AgentsView
            agents={agents}
            ollamaHealthy={ollamaHealthy}
            ollamaModels={ollamaModels}
            onCreateAgent={onCreateAgent as unknown as AgentsViewProps['onCreateAgent']}
            onDeleteAgent={onDeleteAgent as unknown as AgentsViewProps['onDeleteAgent']}
            onUpdateAgent={onUpdateAgent as unknown as AgentsViewProps['onUpdateAgent']}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'ontology':
      return (
        <AlephErrorBoundary key="ontology">
          <OntologyView
            ontologyRaw={ontologyRaw}
            setOntologyRaw={setOntologyRaw}
            onEmerge={onEmerge}
            onSave={onSave}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'data':
      return (
        <AlephErrorBoundary key="data">
          <DataSourcesView
            tasks={ingestionTasks}
            onAddSource={onAddSource}
            onRunTask={onRunTask}
            onViewLogs={onViewLogs}
            onDeleteTask={onDeleteTask}
            taskLogs={taskLogs}
            setTaskLogs={setTaskLogs}
            inline
          />
        </AlephErrorBoundary>
      )
      case 'health':
        return <AlephErrorBoundary key="health"><DataHealthView stats={dataHealthStats} inline /></AlephErrorBoundary>
    case 'skill':
      if ((content.data as Skill | undefined)?.id) {
        return <AlephErrorBoundary key="skill-execute"><SkillExecuteSlideOver skill={content.data as Skill} title={content.title} /></AlephErrorBoundary>
      }
      return (
        <AlephErrorBoundary key="skill">
          <SkillsView
            skills={skills}
            tools={tools}
            onCreateSkill={onCreateSkill as unknown as SkillsViewProps['onCreateSkill']}
            onViewSkillDetail={onViewSkillDetail as unknown as SkillsViewProps['onViewSkillDetail']}
            onDeleteSkill={onDeleteSkill as unknown as SkillsViewProps['onDeleteSkill']}
            onRunSkill={onRunSkill as unknown as SkillsViewProps['onRunSkill']}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'tool':
      if ((content.data as Tool | undefined)?.id) {
        return <AlephErrorBoundary key="tool-execute"><ToolExecuteSlideOver tool={content.data as Tool} title={content.title} /></AlephErrorBoundary>
      }
      return (
        <AlephErrorBoundary key="tool">
          <ToolsView
            tools={tools}
            onCreateTool={onCreateTool as unknown as ToolsViewProps['onCreateTool']}
            onEditTool={onEditTool as unknown as ToolsViewProps['onEditTool']}
            onDeleteTool={onDeleteTool as unknown as ToolsViewProps['onDeleteTool']}
            onExecuteTool={onExecuteTool as unknown as ToolsViewProps['onExecuteTool']}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'component':
      return (
        <AlephErrorBoundary key="component">
          <ComponentsView
            components={registryComponents}
            onUpdateComponentStatus={onUpdateComponentStatus as unknown as ComponentsViewProps['onUpdateComponentStatus']}
            onRegisterComponent={onRegisterComponent as unknown as ComponentsViewProps['onRegisterComponent']}
            onGetComponent={onGetComponent as unknown as ComponentsViewProps['onGetComponent']}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'settings':
      return (
        <AlephErrorBoundary key="settings">
          <SettingsView
            apiKeys={apiKeys}
            notificationChannels={notificationChannels}
            onCreateApiKey={onCreateApiKey}
            onDeleteApiKey={onDeleteApiKey}
            onSendWebhook={onSendWebhook}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'library':
      return (
        <AlephErrorBoundary key="library">
          <LibraryView
            assets={assets}
            onViewAsset={onViewAsset}
            onDeleteAsset={onDeleteAsset}
            selectedAssetContent={selectedAssetContent}
            setSelectedAssetContent={setSelectedAssetContent}
            selectedAssetName={assets.find((a: any) => a.id === selectedAssetId)?.name}
            onGetAssetContent={onGetAssetContent}
            onGeneratePdf={onGeneratePdf}
            onUploadAsset={onUploadAsset}
            selectedAssetId={selectedAssetId}
            inline
          />
        </AlephErrorBoundary>
      )
    case 'predict':
      return <AlephErrorBoundary key="predict"><OracleView inline /></AlephErrorBoundary>
    case 'sandbox': {
      const result = content.data as SandboxResult | undefined
      return <AlephErrorBoundary key="sandbox"><SandboxResultSlideOver result={result} /></AlephErrorBoundary>
    }
     case 'agent-form': {
       const agent = content.data as Agent | undefined
       return <AlephErrorBoundary key="agent-form"><AgentFormSlideOver agent={agent} title={content.title} /></AlephErrorBoundary>
     }
     
     case 'skill-form': {
        const { tools } = content.data as { tools: Tool[] }
        const skill = content.data as Skill | undefined
        return <AlephErrorBoundary key="skill-form"><SkillFormSlideOver skill={skill} tools={tools} title={content.title} /></AlephErrorBoundary>
     }
     
     case 'tool-form': {
        const tool = content.data as Tool | undefined
        return <AlephErrorBoundary key="tool-form"><ToolFormSlideOver tool={tool} title={content.title} /></AlephErrorBoundary>
     }
     
     case 'datasource-form': {
        return <AlephErrorBoundary key="datasource-form"><DataSourceFormSlideOver title={content.title} /></AlephErrorBoundary>
     }
     
       case 'component-form': {
         return <AlephErrorBoundary key="component-form"><ComponentFormSlideOver title={content.title} onClose={() => useStore.getState().setSlideOverContent(null)} /></AlephErrorBoundary>
       }
       
       case 'component-detail': {
         const { componentId } = content.data as { componentId: string }
         return <AlephErrorBoundary key="component-detail"><ComponentDetailSlideOver componentId={componentId} title={content.title} onClose={() => useStore.getState().setSlideOverContent(null)} /></AlephErrorBoundary>
       }
       
       case 'asset': {
         const { assetId } = content.data as { assetId: string }
         return <AlephErrorBoundary key="asset"><AssetDetailSlideOver assetId={assetId} title={content.title} onClose={() => useStore.getState().setSlideOverContent(null)} /></AlephErrorBoundary>
       }
       
       case 'detail': {
         const detailData = content.data as Record<string, unknown>
         return <AlephErrorBoundary key="detail"><DetailSlideOver data={detailData} title={content.title} onClose={() => useStore.getState().setSlideOverContent(null)} /></AlephErrorBoundary>
       }
     
      case 'scenario-comparison': {
        return <AlephErrorBoundary key="scenario-comparison"><ScenarioComparisonView /></AlephErrorBoundary>
      }
     
      default:
       return null
    }
  }

  return (
    <Suspense fallback={<div className="p-4 text-textDim text-xs font-mono">{t('views.loadingGeneric')}</div>}>
      {renderContent()}
    </Suspense>
  )
});
