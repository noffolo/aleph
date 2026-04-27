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
  const store = useStore()
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
  const content = store.inlineContent
  if (!content || !store.showInlinePanel) return null

  const renderView = () => {
    switch (content.type) {
      case 'explore':
        return (
          <InlineErrorBoundary label="ExplorerView">
            <ExplorerView
              availableObjects={store.availableObjects}
              selectedObject={store.selectedObject}
              setSelectedObject={store.setSelectedObject}
            searchQuery={store.searchQuery}
            setSearchQuery={store.setSearchQuery}
            activeView={store.activeView}
            setActiveView={store.setActiveView}
            data={store.searchQuery ? store.globalSearchResults : store.data}
            onRowClick={store.setSelectedRow}
            isLoading={store.isExplorerLoading}
            inline
          />
          </InlineErrorBoundary>
        )
      case 'agent':
        return (
          <AgentsView
            agents={store.agents}
            ollamaHealthy={store.ollamaHealthy}
            ollamaModels={store.ollamaModels}
            onCreateAgent={onCreateAgent as any}
            onDeleteAgent={onDeleteAgent as any}
            onUpdateAgent={onUpdateAgent as any}
            inline
          />
        )
      case 'ontology':
        return (
          <OntologyView
            ontologyRaw={store.ontologyRaw}
            setOntologyRaw={store.setOntologyRaw}
            onEmerge={onEmerge}
            onSave={onSave}
            inline
          />
        )
      case 'data':
        return (
          <DataSourcesView
            tasks={store.ingestionTasks}
            onAddSource={onAddSource}
            onRunTask={onRunTask}
            onViewLogs={onViewLogs}
            onDeleteTask={onDeleteTask}
            taskLogs={store.taskLogs}
            setTaskLogs={store.setTaskLogs}
            inline
          />
        )
      case 'health':
        return <DataHealthView stats={store.dataHealthStats} inline />
      case 'skill':
        return (
          <SkillsView
            skills={store.skills}
            tools={store.tools}
            onCreateSkill={onCreateSkill as any}
            onViewSkillDetail={onViewSkillDetail as any}
            onDeleteSkill={onDeleteSkill as any}
            onRunSkill={onRunSkill as any}
            inline
          />
        )
      case 'tool':
        return (
          <ToolsView
            tools={store.tools}
            onCreateTool={onCreateTool as any}
            onEditTool={onEditTool as any}
            onDeleteTool={onDeleteTool as any}
            onExecuteTool={onExecuteTool as any}
            inline
          />
        )
      case 'component':
        return (
          <ComponentsView
            components={store.registryComponents}
            onUpdateComponentStatus={onUpdateComponentStatus as any}
            onRegisterComponent={onRegisterComponent as any}
            onGetComponent={onGetComponent as any}
            inline
          />
        )
      case 'settings':
        return (
          <SettingsView
            apiKeys={store.apiKeys}
            notificationChannels={store.notificationChannels}
            onCreateApiKey={onCreateApiKey}
            onDeleteApiKey={onDeleteApiKey}
            onSendWebhook={onSendWebhook}
            inline
          />
        )
      case 'library':
        return (
          <LibraryView
            assets={store.assets}
            onViewAsset={onViewAsset}
            onDeleteAsset={onDeleteAsset}
            selectedAssetContent={store.selectedAssetContent}
            setSelectedAssetContent={store.setSelectedAssetContent}
            selectedAssetName={store.assets.find((a: any) => a.id === store.selectedAssetId)?.name}
            onGetAssetContent={onGetAssetContent}
            onGeneratePdf={onGeneratePdf}
            onUploadAsset={onUploadAsset}
            selectedAssetId={store.selectedAssetId}
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
            onClick={() => store.setShowInlinePanel(false)}
            className="text-textMuted hover:text-text text-xs transition-colors px-2 py-1 rounded hover:bg-surface-alt"
          >
             {t('slideOver.close')}
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        <Suspense fallback={<div className="p-4 text-textDim text-xs font-mono">Caricamento vista...</div>}>
          {renderView()}
        </Suspense>
      </div>
    </div>
  )
}

export default InlineRenderer
