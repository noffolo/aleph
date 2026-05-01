import React, { Suspense } from 'react'
import { useStore } from '../../store/useStore'
import { useAppActions } from '../../hooks/useAppActions'
import { useExplorerActions } from '../../hooks/domain/useExplorerActions'
import { useAgentActions } from '../../hooks/domain/useAgentActions'
import { useOntologyActions } from '../../hooks/domain/useOntologyActions'
import { useDataSourceActions } from '../../hooks/domain/useDataSourceActions'
import { useSkillActions } from '../../hooks/domain/useSkillActions'
import { useToolActions } from '../../hooks/domain/useToolActions'
import { useComponentActions } from '../../hooks/domain/useComponentActions'
import { useSettingsActions } from '../../hooks/domain/useSettingsActions'
import { useLibraryActions } from '../../hooks/domain/useLibraryActions'
import { InlineErrorBoundary } from '../InlineErrorBoundary'
import type { ComponentsViewProps } from '../ComponentsView'
import type { SkillsViewProps } from '../SkillsView'
import type { ToolsViewProps } from '../ToolsView'
import type { AgentsViewProps } from '../AgentsView'
import { t } from '../../i18n'

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

export const InlineRenderer: React.FC = () => {
  const inlineContent = useStore(s => s.inlineContent)
  const showInlinePanel = useStore(s => s.showInlinePanel)
  const availableObjects = useStore(s => s.availableObjects)
  const selectedObject = useStore(s => s.selectedObject)
  const searchQuery = useStore(s => s.searchQuery)
  const activeView = useStore(s => s.activeView)
  const globalSearchResults = useStore(s => s.globalSearchResults)
  const explorerData = useStore(s => s.data)
  const selectedRow = useStore(s => s.selectedRow)
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
  const setSelectedObject = useStore(s => s.setSelectedObject)
  const setSearchQuery = useStore(s => s.setSearchQuery)
  const setActiveView = useStore(s => s.setActiveView)
  const { loadProjectData } = useAppActions()
  const {
    onCreateAgent, onDeleteAgent, onUpdateAgent,
  } = useAgentActions(loadProjectData)
  const {
    onEmerge, onSave,
  } = useOntologyActions(loadProjectData)
  const {
    onAddSource, onRunTask, onViewLogs, onDeleteTask,
  } = useDataSourceActions(loadProjectData)
  const {
    onCreateSkill, onViewSkillDetail, onDeleteSkill, onRunSkill,
  } = useSkillActions(loadProjectData)
  const {
    onCreateTool, onEditTool, onDeleteTool, onExecuteTool,
  } = useToolActions(loadProjectData)
  const {
    onUpdateComponentStatus, onRegisterComponent, onGetComponent,
  } = useComponentActions()
  const {
    onCreateApiKey, onDeleteApiKey, onSendWebhook,
  } = useSettingsActions()
  const {
    onViewAsset, onDeleteAsset, onGetAssetContent, onGeneratePdf, onUploadAsset,
  } = useLibraryActions(loadProjectData)
  const content = inlineContent
  if (!content || !showInlinePanel) return null

  const renderView = () => {
    switch (content.type) {
      case 'explore':
        return (
          <InlineErrorBoundary label="ExplorerView">
            <ExplorerView
              availableObjects={availableObjects}
              selectedObject={selectedObject}
              setSelectedObject={setSelectedObject}
            searchQuery={searchQuery}
            setSearchQuery={setSearchQuery}
            activeView={activeView}
            setActiveView={setActiveView}
            data={searchQuery ? globalSearchResults : explorerData}
            onRowClick={setSelectedRow}
            isLoading={isExplorerLoading}
            inline
          />
          </InlineErrorBoundary>
        )
      case 'agent':
        return (
          <AgentsView
            agents={agents}
            ollamaHealthy={ollamaHealthy}
            ollamaModels={ollamaModels}
            onCreateAgent={onCreateAgent as unknown as AgentsViewProps['onCreateAgent']}
            onDeleteAgent={onDeleteAgent as unknown as AgentsViewProps['onDeleteAgent']}
            onUpdateAgent={onUpdateAgent as unknown as AgentsViewProps['onUpdateAgent']}
            inline
          />
        )
      case 'ontology':
        return (
          <OntologyView
            ontologyRaw={ontologyRaw}
            setOntologyRaw={setOntologyRaw}
            onEmerge={onEmerge}
            onSave={onSave}
            inline
          />
        )
      case 'data':
        return (
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
        )
      case 'health':
        return <DataHealthView stats={dataHealthStats} inline />
      case 'skill':
        return (
          <SkillsView
            skills={skills}
            tools={tools}
            onCreateSkill={onCreateSkill as unknown as SkillsViewProps['onCreateSkill']}
            onViewSkillDetail={onViewSkillDetail as unknown as SkillsViewProps['onViewSkillDetail']}
            onDeleteSkill={onDeleteSkill as unknown as SkillsViewProps['onDeleteSkill']}
            onRunSkill={onRunSkill as unknown as SkillsViewProps['onRunSkill']}
            inline
          />
        )
      case 'tool':
        return (
          <ToolsView
            tools={tools}
            onCreateTool={onCreateTool as unknown as ToolsViewProps['onCreateTool']}
            onEditTool={onEditTool as unknown as ToolsViewProps['onEditTool']}
            onDeleteTool={onDeleteTool as unknown as ToolsViewProps['onDeleteTool']}
            onExecuteTool={onExecuteTool as unknown as ToolsViewProps['onExecuteTool']}
            inline
          />
        )
      case 'component':
        return (
          <ComponentsView
            components={registryComponents}
            onUpdateComponentStatus={onUpdateComponentStatus as unknown as ComponentsViewProps['onUpdateComponentStatus']}
            onRegisterComponent={onRegisterComponent as unknown as ComponentsViewProps['onRegisterComponent']}
            onGetComponent={onGetComponent as unknown as ComponentsViewProps['onGetComponent']}
            inline
          />
        )
      case 'settings':
        return (
          <SettingsView
            apiKeys={apiKeys}
            notificationChannels={notificationChannels}
            onCreateApiKey={onCreateApiKey}
            onDeleteApiKey={onDeleteApiKey}
            onSendWebhook={onSendWebhook}
            inline
          />
        )
      case 'library':
        return (
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
        )
      default:
        return null
    }
  }

  return (
    <div className="flex flex-col h-full bg-surface rounded-lg border border-border overflow-hidden animate-fade-in">
      <div className="h-9 flex items-center justify-between px-4 border-b border-border shrink-0">
        <span className="text-primary text-xs font-bold">{ '📄 ' + content.title.toUpperCase() }</span>
        <div className="flex items-center gap-2">
          <button
            onClick={() => useStore.getState().setShowInlinePanel(false)}
            aria-label={t('slideOver.close')}
            className="text-textMuted hover:text-text text-xs transition-colors px-2 py-1 rounded hover:bg-surface-alt focus:ring-2 focus:ring-primary"
          >
             {t('slideOver.close')}
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        <Suspense fallback={<div className="p-4 text-textDim text-xs font-mono">{t('views.loading')}</div>}>
          {renderView()}
        </Suspense>
      </div>
    </div>
  )
}

export default InlineRenderer
