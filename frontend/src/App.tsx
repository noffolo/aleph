import { useState, useEffect } from 'react'
import { Sidebar } from './components/Sidebar'
import { WorkspaceOnboarding } from './components/WorkspaceOnboarding'
import { ExplorerView } from './components/ExplorerView'
import { OracleView } from './components/OracleView'
import { AgentsView } from './components/AgentsView'
import { OntologyView } from './components/OntologyView'
import { DataSourcesView } from './components/DataSourcesView'
import { CopilotView } from './components/CopilotView'
import { DetailPanel } from './components/DetailPanel'
import { LibraryView } from './components/LibraryView'
import { CommandPalette } from './components/CommandPalette'
import { SetupWizard } from './components/SetupWizard'
import { AlephErrorBoundary } from './components/AlephErrorBoundary'
import { useStore } from './store/useStore'
import { 
    projectClient, 
    queryClient, 
    agentClient, 
    ingestionClient, 
    libraryClient, 
    authClient,
    nlpClient
} from "./api/factory"
import { setApiKey } from "./api/client"

function App() {
  const { activeTab, setActiveTab, selectedObject, setSelectedObject, projectID, setProjectContext } = useStore()
  const [data, setData] = useState<any>(null)
  const [selectedRow, setSelectedRow] = useState<any>(null)
  const [projects, setProjects] = useState<any[]>([])
  const [agents, setAgents] = useState<any[]>([])
  const [selectedAgent, setSelectedAgent] = useState<string>('')
  const [ingestionTasks, setIngestionTasks] = useState<any[]>([])
  const [ontologyRaw, setOntologyRaw] = useState<string>('')
  const [chat, setChat] = useState<any[]>([])
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [showOnboarding, setShowOnboarding] = useState(true)
  const [showWizard, setShowWizard] = useState(false)
  const [assets, setAssets] = useState<any[]>([])
  const [selectedAssetContent, setSelectedAssetContent] = useState<string | null>(null)
  const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState(false)
  const [isExplorerLoading, setIsExplorerLoading] = useState(false)

  useEffect(() => {
    projectClient.listProjects({}).then((res: any) => setProjects(res.projects));
  }, [])

  const onSend = async () => {
    if (!input || isStreaming) return;
    const msg = { role: 'user', content: input };
    setChat(prev => [...prev, msg]);
    setInput('');
    setIsStreaming(true);
    let assistantMsg = { role: 'assistant', content: '', toolCall: '' };
    setChat(prev => [...prev, assistantMsg]);
    try {
      // Usiamo il metodo corretto dal proto: streamPredictions
      const stream = nlpClient.streamPredictions({ contextId: projectID, ontologyQuery: input });
      for await (const res of stream) {
        assistantMsg.content += res.explanation; // Usiamo explanation come contenuto
        setChat(prev => [...prev.slice(0, -1), { ...assistantMsg }]);
      }
    } finally { setIsStreaming(false); }
  }

  if (showWizard) return <SetupWizard 
    onCreateProject={async (n: string) => { const r = await projectClient.createProject({ id: n.toLowerCase(), name: n }); return r.project!.id; }}
    onCreateApiKey={async (pid: string, l: string) => { const r = await authClient.createApiKey({ projectId: pid, label: l }); return r.key!.key; }}
    onComplete={(pid, key) => { setApiKey(key); setProjectContext(pid, key); setShowWizard(false); setShowOnboarding(false); setActiveTab('Intelligence'); }}
  />

  if (showOnboarding) return <WorkspaceOnboarding 
    projects={projects} 
    onSelectProject={(id, key) => { setApiKey(key); setProjectContext(id, key); setShowOnboarding(false); setActiveTab('Intelligence'); }}
    onDeleteProject={(id) => projectClient.deleteProject({id: id}).then(() => projectClient.listProjects({}).then((res: any) => setProjects(res.projects)))}
    onCreateProject={() => setShowWizard(true)}
  />

  return (
    <div className="flex h-screen bg-white text-gray-900 font-sans overflow-hidden">
      <AlephErrorBoundary>
        <CommandPalette 
          isOpen={isCommandPaletteOpen} onClose={() => setIsCommandPaletteOpen(false)} 
          availableObjects={[]} projects={projects}
          onSelectProject={(id) => { setProjectContext(id, ''); setShowOnboarding(false); setActiveTab('Intelligence'); }}
          onSelectObject={(obj) => { setSelectedObject(obj); setActiveTab('Intelligence'); }}
          setActiveTab={setActiveTab}
        />
        <Sidebar activeTab={activeTab} setActiveTab={setActiveTab} projectID={projectID} onShowOnboarding={() => setShowOnboarding(true)} />
        <div className="flex-1 flex flex-col overflow-hidden relative">
          <header className="h-16 border-b flex items-center px-8 justify-between bg-white z-10 shrink-0 shadow-sm">
            <h1 className="text-xl font-black tracking-tighter text-blue-900 uppercase italic">{activeTab}</h1>
          </header>
          <main className="flex-1 overflow-auto p-10 bg-gray-50/10">
             {activeTab === 'Explorer' && <ExplorerView availableObjects={[]} selectedObject={selectedObject} setSelectedObject={setSelectedObject} searchQuery={""} setSearchQuery={() => {}} activeView={'table'} setActiveView={() => {}} data={data} onRowClick={setSelectedRow} isLoading={isExplorerLoading} />}
             <DetailPanel selectedRow={selectedRow} onClose={() => setSelectedRow(null)} />
          </main>
        </div>
      </AlephErrorBoundary>
    </div>
  )
}

export default App
