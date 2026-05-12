import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

// --- Shared mock fns ---
const mockSetSlideOverContent = vi.fn()

// --- Mock state factory ---
const createMockState = (overrides: Record<string, unknown> = {}) => ({
  slideOverContent: null,
  availableObjects: [],
  selectedObject: 'proj-1',
  setSelectedObject: vi.fn(),
  searchQuery: '',
  setSearchQuery: vi.fn(),
  activeView: 'table',
  setActiveView: vi.fn(),
  globalSearchResults: [],
  data: [],
  setSelectedRow: vi.fn(),
  isExplorerLoading: false,
  agents: [],
  ollamaHealthy: false,
  ollamaModels: [],
  ontologyRaw: '',
  setOntologyRaw: vi.fn(),
  ingestionTasks: [],
  taskLogs: [],
  setTaskLogs: vi.fn(),
  dataHealthStats: { totalRows: 0, nullCount: 0, duplicateCount: 0 },
  skills: [],
  tools: [],
  registryComponents: [],
  apiKeys: [],
  notificationChannels: [],
  assets: [],
  selectedAssetContent: null,
  setSelectedAssetContent: vi.fn(),
  selectedAssetId: null,
  ...overrides,
})

// Use a mutable ref so setSlideOverContent can change what useStore returns
let currentState = createMockState()

vi.mock('../../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (s: Record<string, unknown>) => unknown) => {
      if (typeof selector === 'function') {
        return selector(currentState)
      }
      return currentState
    }),
    {
      getState: vi.fn(() => ({
        ...currentState,
        setSlideOverContent: mockSetSlideOverContent,
      })),
      subscribe: vi.fn(() => vi.fn()),
    },
  ),
}))

// --- Mock domain hooks ---
const mockDomainActions = () => ({
  loadProjectData: vi.fn(),
})

const mockAgentActions = {
  onCreateAgent: vi.fn(),
  onDeleteAgent: vi.fn(),
  onUpdateAgent: vi.fn(),
}

const mockOntologyActions = {
  onEmerge: vi.fn(),
  onSave: vi.fn(),
}

const mockDataSourceActions = {
  onAddSource: vi.fn(),
  onRunTask: vi.fn(),
  onViewLogs: vi.fn(),
  onDeleteTask: vi.fn(),
}

const mockSkillActions = {
  onCreateSkill: vi.fn(),
  onViewSkillDetail: vi.fn(),
  onDeleteSkill: vi.fn(),
  onRunSkill: vi.fn(),
}

const mockToolActions = {
  onCreateTool: vi.fn(),
  onEditTool: vi.fn(),
  onDeleteTool: vi.fn(),
  onExecuteTool: vi.fn(),
}

const mockComponentActions = {
  onUpdateComponentStatus: vi.fn(),
  onRegisterComponent: vi.fn(),
  onGetComponent: vi.fn(),
}

const mockSettingsActions = {
  onCreateApiKey: vi.fn(),
  onDeleteApiKey: vi.fn(),
  onSendWebhook: vi.fn(),
}

const mockLibraryActions = {
  onViewAsset: vi.fn(),
  onDeleteAsset: vi.fn(),
  onGetAssetContent: vi.fn(),
  onGeneratePdf: vi.fn(),
  onUploadAsset: vi.fn(),
}

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => mockDomainActions(),
}))

vi.mock('../../../hooks/domain/useAgentActions', () => ({
  useAgentActions: () => mockAgentActions,
}))

vi.mock('../../../hooks/domain/useOntologyActions', () => ({
  useOntologyActions: () => mockOntologyActions,
}))

vi.mock('../../../hooks/domain/useDataSourceActions', () => ({
  useDataSourceActions: () => mockDataSourceActions,
}))

vi.mock('../../../hooks/domain/useSkillActions', () => ({
  useSkillActions: () => mockSkillActions,
}))

vi.mock('../../../hooks/domain/useToolActions', () => ({
  useToolActions: () => mockToolActions,
}))

vi.mock('../../../hooks/domain/useComponentActions', () => ({
  useComponentActions: () => mockComponentActions,
}))

vi.mock('../../../hooks/domain/useSettingsActions', () => ({
  useSettingsActions: () => mockSettingsActions,
}))

vi.mock('../../../hooks/domain/useLibraryActions', () => ({
  useLibraryActions: () => mockLibraryActions,
}))

// --- Mock i18n ---
vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'generic.loading': 'Caricamento...',
    }
    return map[key] ?? key
  },
}))

// --- Mock AlephErrorBoundary (passthrough) ---
vi.mock('../../AlephErrorBoundary', () => ({
  AlephErrorBoundary: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

// --- Mock SkeletonLoader ---
vi.mock('../../SkeletonLoader', () => ({
  SkeletonLoader: ({ rows, cols }: { rows: number; cols: number }) => (
    <div data-testid="skeleton-loader">Skeleton: {rows}x{cols}</div>
  ),
}))

// --- Stub component factory (renders a data-testid + data-prop-* attrs) ---
function stubComponent(testId: string) {
  const Stub: React.FC<Record<string, unknown>> = (props) => {
    const dataAttrs: Record<string, string> = {}
    Object.entries(props).forEach(([key, value]) => {
      if (key === 'children') return
      const safeKey = `data-prop-${key}`
      if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
        dataAttrs[safeKey] = String(value)
      }
    })
    return <div data-testid={testId} {...dataAttrs} />
  }
  Stub.displayName = `Stub(${testId})`
  return Stub
}

// --- Mock all lazy-loaded modules ---
vi.mock('../../ExplorerView', () => ({ ExplorerView: stubComponent('explorer-view') }))
vi.mock('../../AgentsView', () => ({ AgentsView: stubComponent('agents-view') }))
vi.mock('../../OntologyView', () => ({ OntologyView: stubComponent('ontology-view') }))
vi.mock('../../DataSourcesView', () => ({ DataSourcesView: stubComponent('datasources-view') }))
vi.mock('../../DataHealthView', () => ({ DataHealthView: stubComponent('datahealth-view') }))
vi.mock('../../SettingsView', () => ({ SettingsView: stubComponent('settings-view') }))
vi.mock('../../ComponentsView', () => ({ ComponentsView: stubComponent('components-view') }))
vi.mock('../../SkillsView', () => ({ SkillsView: stubComponent('skills-view') }))
vi.mock('../../ToolsView', () => ({ ToolsView: stubComponent('tools-view') }))
vi.mock('../../LibraryView', () => ({ LibraryView: stubComponent('library-view') }))
vi.mock('../../OracleView', () => ({ OracleView: stubComponent('oracle-view') }))
vi.mock('../../ToolIntelligenceView', () => ({ default: stubComponent('tool-intelligence-view') }))

// --- Mock direct-import form components ---
vi.mock('../../forms/AgentFormSlideOver', () => ({
  AgentFormSlideOver: stubComponent('agent-form-slideover'),
}))
vi.mock('../../forms/SkillFormSlideOver', () => ({
  SkillFormSlideOver: stubComponent('skill-form-slideover'),
}))
vi.mock('../../forms/ToolFormSlideOver', () => ({
  ToolFormSlideOver: stubComponent('tool-form-slideover'),
}))
vi.mock('../../forms/DataSourceFormSlideOver', () => ({
  DataSourceFormSlideOver: stubComponent('datasource-form-slideover'),
}))
vi.mock('../../forms/SkillExecuteSlideOver', () => ({
  SkillExecuteSlideOver: stubComponent('skill-execute-slideover'),
}))
vi.mock('../../forms/ToolExecuteSlideOver', () => ({
  ToolExecuteSlideOver: stubComponent('tool-execute-slideover'),
}))
vi.mock('../../forms/SandboxResultSlideOver', () => ({
  SandboxResultSlideOver: stubComponent('sandbox-result-slideover'),
}))
vi.mock('../../forms/ComponentFormSlideOver', () => ({
  ComponentFormSlideOver: stubComponent('component-form-slideover'),
}))
vi.mock('../../forms/ComponentDetailSlideOver', () => ({
  ComponentDetailSlideOver: stubComponent('component-detail-slideover'),
}))
vi.mock('../../forms/AssetDetailSlideOver', () => ({
  AssetDetailSlideOver: stubComponent('asset-detail-slideover'),
}))
vi.mock('../../forms/DetailSlideOver', () => ({
  DetailSlideOver: stubComponent('detail-slideover'),
}))
vi.mock('../../../views/ScenarioComparisonView', () => ({
  ScenarioComparisonView: stubComponent('scenario-comparison-view'),
}))

// --- Import AFTER all mocks ---
import { SlideOverContent } from '../SlideOverContent'

/**
 * Mutates currentState so that on the next render,
 * useStore selector sees the desired slideOverContent and data.
 */
function setSlideOverContent(content: Record<string, unknown> | null) {
  currentState = createMockState({
    slideOverContent: content,
    agents: [{ id: 'agent-1', name: 'Test Agent', model: 'gpt-4', provider: 'openai' }],
    skills: [{ id: 'skill-1', name: 'Test Skill' }],
    tools: [{ id: 'tool-1', name: 'Test Tool' }],
    assets: [{ id: 'asset-1', name: 'report.pdf' }],
    registryComponents: [{ id: 'comp-1', name: 'Test Component' }],
    apiKeys: [{ id: 'key-1', name: 'Default Key', key: 'sk-test' }],
    notificationChannels: [{ id: 'ch-1', type: 'webhook', url: 'https://example.com' }],
  })
}

describe('SlideOverContent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // reset to default (null slideOverContent)
    currentState = createMockState()
  })

  // === Null content ===

  it('returns null when slideOverContent is null', () => {
    const { container } = render(<SlideOverContent />)
    expect(container.innerHTML).toBe('')
  })

  // === Lazy-loaded view: ExplorerView ===

  it('renders ExplorerView for type "explore" with inline prop', async () => {
    setSlideOverContent({ type: 'explore' })
    render(<SlideOverContent />)
    await waitFor(() => {
      const el = screen.getByTestId('explorer-view')
      expect(el).toBeInTheDocument()
      expect(el.getAttribute('data-prop-inline')).toBe('true')
    })
  })

  // === Lazy-loaded view: AgentsView ===

  it('renders AgentsView for type "agent" with agents and inline props', async () => {
    setSlideOverContent({ type: 'agent' })
    render(<SlideOverContent />)
    await waitFor(() => {
      const el = screen.getByTestId('agents-view')
      expect(el).toBeInTheDocument()
      expect(el.getAttribute('data-prop-inline')).toBe('true')
    })
  })

  // === Lazy-loaded view: ToolsView (no tool ID) ===

  it('renders ToolsView for type "tool" when data has no id', async () => {
    setSlideOverContent({ type: 'tool' })
    render(<SlideOverContent />)
    await waitFor(() => {
      const el = screen.getByTestId('tools-view')
      expect(el).toBeInTheDocument()
      expect(el.getAttribute('data-prop-inline')).toBe('true')
    })
  })

  // === Direct-import: ToolExecuteSlideOver (tool with ID) ===

  it('renders ToolExecuteSlideOver for type "tool" when data has id', () => {
    setSlideOverContent({ type: 'tool', data: { id: 'tool-1' }, title: 'Run Tool' })
    render(<SlideOverContent />)
    const el = screen.getByTestId('tool-execute-slideover')
    expect(el).toBeInTheDocument()
  })

  // === Lazy-loaded view: SettingsView ===

  it('renders SettingsView for type "settings"', async () => {
    setSlideOverContent({ type: 'settings' })
    render(<SlideOverContent />)
    await waitFor(() => {
      const el = screen.getByTestId('settings-view')
      expect(el).toBeInTheDocument()
      expect(el.getAttribute('data-prop-inline')).toBe('true')
    })
  })

  // === Lazy-loaded view: LibraryView ===

  it('renders LibraryView for type "library"', async () => {
    setSlideOverContent({ type: 'library' })
    render(<SlideOverContent />)
    await waitFor(() => {
      const el = screen.getByTestId('library-view')
      expect(el).toBeInTheDocument()
      expect(el.getAttribute('data-prop-inline')).toBe('true')
    })
  })

  // === Lazy-loaded view: OracleView ===

  it('renders OracleView for type "predict"', async () => {
    setSlideOverContent({ type: 'predict' })
    render(<SlideOverContent />)
    await waitFor(() => {
      const el = screen.getByTestId('oracle-view')
      expect(el).toBeInTheDocument()
      expect(el.getAttribute('data-prop-inline')).toBe('true')
    })
  })

  // === Confirm dialog (inline, no lazy) ===

  it('renders confirmation dialog with Annulla and Conferma buttons for type "confirm"', () => {
    setSlideOverContent({
      type: 'confirm',
      title: 'Delete Item',
      data: 'Are you sure you want to delete this item?',
    })
    render(<SlideOverContent />)
    expect(screen.getByText('Delete Item')).toBeInTheDocument()
    expect(screen.getByText('Are you sure you want to delete this item?')).toBeInTheDocument()
    expect(screen.getByText('Annulla')).toBeInTheDocument()
    expect(screen.getByText('Conferma')).toBeInTheDocument()
  })

  // === Default case (unknown type) ===

  it('returns null for unknown type (default case)', () => {
    setSlideOverContent({ type: 'nonexistent-type' })
    render(<SlideOverContent />)
    // No lazy component resolves for unknown types; Suspense contains nothing
    expect(screen.queryByTestId('explorer-view')).not.toBeInTheDocument()
    expect(screen.queryByTestId('agents-view')).not.toBeInTheDocument()
  })

  // === Direct-import: SandboxResultSlideOver ===

  it('renders SandboxResultSlideOver for type "sandbox"', () => {
    setSlideOverContent({ type: 'sandbox', data: { stdout: 'OK', stderr: '', exitCode: 0 } })
    render(<SlideOverContent />)
    const el = screen.getByTestId('sandbox-result-slideover')
    expect(el).toBeInTheDocument()
  })
})
