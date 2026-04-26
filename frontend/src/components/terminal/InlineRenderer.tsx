import React, { Suspense } from 'react'
import { useStore } from '../../store/useStore'
import { useViewActions } from '../../hooks/useViewActions'
import { InlineErrorBoundary } from '../InlineErrorBoundary'

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
  const actions = useViewActions()
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
            onCreateAgent={actions.agentsActions.onCreateAgent}
            onDeleteAgent={actions.agentsActions.onDeleteAgent}
            onUpdateAgent={actions.agentsActions.onUpdateAgent}
            inline
          />
        )
      case 'ontology':
        return (
          <OntologyView
            ontologyRaw={store.ontologyRaw}
            setOntologyRaw={store.setOntologyRaw}
            onEmerge={actions.ontologyActions.onEmerge}
            onSave={actions.ontologyActions.onSave}
            inline
          />
        )
      case 'data':
        return (
          <DataSourcesView
            tasks={store.ingestionTasks}
            onAddSource={actions.dataSourcesActions.onAddSource}
            onRunTask={actions.dataSourcesActions.onRunTask}
            onViewLogs={actions.dataSourcesActions.onViewLogs}
            onDeleteTask={actions.dataSourcesActions.onDeleteTask}
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
            onCreateSkill={actions.skillsActions.onCreateSkill}
            onViewSkillDetail={actions.skillsActions.onViewSkillDetail}
            onDeleteSkill={actions.skillsActions.onDeleteSkill}
            onRunSkill={actions.skillsActions.onRunSkill}
            inline
          />
        )
      case 'tool':
        return (
          <ToolsView
            tools={store.tools}
            onCreateTool={actions.toolsActions.onCreateTool}
            onEditTool={actions.toolsActions.onEditTool}
            onDeleteTool={actions.toolsActions.onDeleteTool}
            onExecuteTool={actions.toolsActions.onExecuteTool}
            inline
          />
        )
      case 'component':
        return (
          <ComponentsView
            components={store.registryComponents}
            onUpdateComponentStatus={actions.componentsActions.onUpdateComponentStatus}
            onRegisterComponent={actions.componentsActions.onRegisterComponent}
            onGetComponent={actions.componentsActions.onGetComponent}
            inline
          />
        )
      case 'settings':
        return (
          <SettingsView
            apiKeys={store.apiKeys}
            notificationChannels={store.notificationChannels}
            onCreateApiKey={actions.settingsActions.onCreateApiKey}
            onDeleteApiKey={actions.settingsActions.onDeleteApiKey}
            onSendWebhook={actions.settingsActions.onSendWebhook}
            inline
          />
        )
      case 'library':
        return (
          <LibraryView
            assets={store.assets}
            onViewAsset={actions.libraryActions.onViewAsset}
            onDeleteAsset={actions.libraryActions.onDeleteAsset}
            selectedAssetContent={store.selectedAssetContent}
            setSelectedAssetContent={store.setSelectedAssetContent}
            selectedAssetName={store.assets.find((a: any) => a.id === store.selectedAssetId)?.name}
            onGetAssetContent={actions.libraryActions.onGetAssetContent}
            onGeneratePdf={actions.libraryActions.onGeneratePdf}
            onUploadAsset={actions.libraryActions.onUploadAsset}
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
             CHIUDI
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
