import React from 'react'
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

// --- Store mock state (mutable ref for per-test overrides) ---

const mockSetLastError = vi.fn()
const mockSetIsCommandPaletteOpen = vi.fn()
const mockSetSlideOverContent = vi.fn()
const mockSetShowOnboarding = vi.fn()
const mockSetProjectContext = vi.fn()
const mockSetShowWizard = vi.fn()
const mockSetProjects = vi.fn()
const mockSetIsExplorerLoading = vi.fn()
const mockSetData = vi.fn()
const mockSetDataHealthStats = vi.fn()
const mockSetMessages = vi.fn()
const mockSetSelectedObject = vi.fn()
const mockHandleError = vi.fn()
const mockLoadProjectData = vi.fn()
const mockOnSend = vi.fn()
const mockOnConfirmAction = vi.fn()

const storeStateRef: { current: Record<string, unknown> } = {
  current: {
    setLastError: mockSetLastError,
    setIsCommandPaletteOpen: mockSetIsCommandPaletteOpen,
    setSlideOverContent: mockSetSlideOverContent,
    setShowOnboarding: mockSetShowOnboarding,
    setProjectContext: mockSetProjectContext,
    setShowWizard: mockSetShowWizard,
    setProjects: mockSetProjects,
    setIsExplorerLoading: mockSetIsExplorerLoading,
    setData: mockSetData,
    setDataHealthStats: mockSetDataHealthStats,
    setMessages: mockSetMessages,
    setSelectedObject: mockSetSelectedObject,
    projects: [],
    projectID: '',
    selectedObject: null,
    selectedAgent: null,
    showWizard: false,
    showOnboarding: false,
    isCommandPaletteOpen: false,
    availableObjects: [],
    lastError: null as string | null,
    slideOverContent: null as { type: string; title: string; data?: unknown } | null,
    currentScene: 'terminal' as string,
    ollamaHealthy: false,
    nlpHealthy: false,
    inputMode: 'command' as string,
    messages: [],
    expandedSections: {} as Record<string, boolean>,
    toggleSection: vi.fn(),
  },
}

vi.mock('../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') return selector(storeStateRef.current)
      return storeStateRef.current
    }),
    {
      getState: vi.fn(() => storeStateRef.current),
      subscribe: vi.fn(() => vi.fn()),
    },
  ),
}))

vi.mock('../hooks/useAppActions', () => ({
  useAppActions: () => ({
    handleError: mockHandleError,
    loadProjectData: mockLoadProjectData,
    onSend: mockOnSend,
    onConfirmAction: mockOnConfirmAction,
  }),
  handleError: mockHandleError,
}))

vi.mock('../i18n', () => ({
  t: (key: string) => key,
}))

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['', vi.fn()] as const),
}))

vi.mock('lucide-react', () => ({
  Search: () => null,
  Zap: () => null,
  Settings: () => null,
  Activity: () => null,
  Database: () => null,
  Code: () => null,
  Terminal: () => null,
  ChevronDown: () => null,
  ChevronRight: () => null,
  X: () => null,
  Plus: () => null,
  Trash2: () => null,
  Play: () => null,
  Edit: () => null,
  Bot: () => null,
  Brain: () => null,
  Book: () => null,
  FileText: () => null,
  Download: () => null,
  Upload: () => null,
  Wifi: () => null,
  WifiOff: () => null,
}))

vi.mock('../api/factory', () => ({
  projectClient: {
    listProjects: vi.fn(() => new Promise<{ projects: unknown[] }>(() => {})),
    createProject: vi.fn(() => Promise.resolve({ project: { id: 'test-proj' } })),
    deleteProject: vi.fn(() => Promise.resolve()),
  },
  authClient: {
    createApiKey: vi.fn(() => Promise.resolve({ key: { key: 'test-key' } })),
  },
  queryClient: {
    executeQuery: vi.fn(() => new Promise(() => {})),
    getDataStats: vi.fn(() => new Promise(() => {})),
    getChatHistory: vi.fn(() => new Promise(() => {})),
  },
  agentClient: {},
  ingestionClient: {},
  libraryClient: {},
  skillClient: {},
  toolClient: {},
  nlpClient: {},
  registryClient: {},
  sandboxClient: {},
  notificationClient: {},
}))

vi.mock('../api/client', () => ({
  createSession: vi.fn(() => Promise.resolve()),
}))

vi.mock('../hooks/NavigationStateSync', () => ({
  NavigationStateSync: () => null,
}))

vi.mock('../components/SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
  SkeletonList: () => <div data-testid="skeleton-list">Loading...</div>,
}))

// --- Component mocks ---

vi.mock('../components/Sidebar', () => ({
  Sidebar: ({ projectID, onShowOnboarding }: { projectID: string; onShowOnboarding: () => void }) => (
    <nav data-testid="sidebar" data-projectid={projectID}>
      <button data-testid="show-onboarding-btn" onClick={onShowOnboarding}>Onboard</button>
    </nav>
  ),
}))

vi.mock('../components/CommandPalette', () => ({
  CommandPalette: ({ isOpen, onClose, availableObjects, projects, onSelectProject, onSelectObject }: {
    isOpen: boolean
    onClose: () => void
    availableObjects: string[]
    projects: { id: string; name: string }[]
    onSelectProject: (id: string) => void
    onSelectObject: (name: string) => void
  }) => (
    <div data-testid="command-palette" data-isopen={isOpen ? 'true' : 'false'}>
      <button data-testid="cp-close" onClick={onClose}>Close</button>
      {projects.map(p => (
        <button key={p.id} data-testid={`cp-project-${p.id}`} onClick={() => onSelectProject(p.id)}>{p.name}</button>
      ))}
      {availableObjects.map(o => (
        <button key={o} data-testid={`cp-object-${o}`} onClick={() => onSelectObject(o)}>{o}</button>
      ))}
    </div>
  ),
}))

vi.mock('../components/AlephErrorBoundary', () => ({
  AlephErrorBoundary: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock('../components/terminal/SlideOverPanel', () => ({
  SlideOverPanel: ({ isOpen, onClose, title, children }: {
    isOpen: boolean
    onClose: () => void
    title: string
    children: React.ReactNode
  }) => (
    <div data-testid="slide-over-panel" data-isopen={isOpen ? 'true' : 'false'}>
      <h2>{title}</h2>
      <button data-testid="slide-over-close" onClick={onClose}>Close</button>
      {children}
    </div>
  ),
}))

vi.mock('../components/terminal', () => ({
  StatusBar: ({ projectID, ollamaHealthy, nlpHealthy }: {
    projectID: string
    ollamaHealthy: boolean
    nlpHealthy: boolean
  }) => (
    <div data-testid="status-bar" data-projectid={projectID} data-ollama={ollamaHealthy ? 'true' : 'false'} data-nlp={nlpHealthy ? 'true' : 'false'}>
      StatusBar
    </div>
  ),
}))

vi.mock('../components/Toast', () => ({
  ToastContainer: () => <div data-testid="toast-container" />,
}))

// --- Lazy component mocks (must export the named component that .then() extracts) ---

vi.mock('../scenes/SceneSelector', () => {
  const SceneSelector = () => <div data-testid="scene-selector">SceneSelector</div>
  return { SceneSelector, default: SceneSelector }
})

vi.mock('../components/terminal/SlideOverContent', () => {
  const SlideOverContent = () => <div data-testid="slide-over-content">SlideOverContent</div>
  return { SlideOverContent, default: SlideOverContent }
})

vi.mock('../components/SetupWizard', () => {
  const SetupWizard = ({ onComplete }: {
    onLogin: (key: string) => Promise<void>
    onCreateProject: (name: string) => Promise<string>
    onComplete: (pid: string, key: string) => Promise<void>
    onCreateApiKey: (pid: string, label: string) => Promise<string>
  }) => (
    <div data-testid="setup-wizard">
      <button data-testid="wizard-complete" onClick={() => onComplete('proj-1', 'key-1')}>Complete</button>
    </div>
  )
  return { SetupWizard, default: SetupWizard }
})

vi.mock('../components/WorkspaceOnboarding', () => {
  const WorkspaceOnboarding = ({ onCreateProject }: {
    projects: { id: string; name: string }[]
    onSelectProject: (id: string, key: string) => Promise<void>
    onDeleteProject: (id: string, key: string) => Promise<void>
    onCreateProject: () => void
  }) => (
    <div data-testid="workspace-onboarding">
      <button data-testid="onboard-create" onClick={onCreateProject}>New Project</button>
    </div>
  )
  return { WorkspaceOnboarding, default: WorkspaceOnboarding }
})

// --- Test helpers ---

function setStoreState(overrides: Record<string, unknown>) {
  Object.assign(storeStateRef.current, overrides)
}

function resetStoreState() {
  storeStateRef.current = {
    setLastError: mockSetLastError,
    setIsCommandPaletteOpen: mockSetIsCommandPaletteOpen,
    setSlideOverContent: mockSetSlideOverContent,
    setShowOnboarding: mockSetShowOnboarding,
    setProjectContext: mockSetProjectContext,
    setShowWizard: mockSetShowWizard,
    setProjects: mockSetProjects,
    setIsExplorerLoading: mockSetIsExplorerLoading,
    setData: mockSetData,
    setDataHealthStats: mockSetDataHealthStats,
    setMessages: mockSetMessages,
    setSelectedObject: mockSetSelectedObject,
    projects: [],
    projectID: '',
    selectedObject: null,
    selectedAgent: null,
    showWizard: false,
    showOnboarding: false,
    isCommandPaletteOpen: false,
    availableObjects: [],
    lastError: null,
    slideOverContent: null,
    currentScene: 'terminal',
    ollamaHealthy: false,
    nlpHealthy: false,
    inputMode: 'command',
    messages: [],
    expandedSections: {},
    toggleSection: vi.fn(),
  }
}

async function renderApp() {
  const { default: App } = await import('../App')
  let result: ReturnType<typeof render>
  await act(async () => {
    result = render(<App />)
  })
  return result!
}

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetStoreState()
  })

  it('renders main layout without crashing', async () => {
    await renderApp()

    await waitFor(() => {
      expect(screen.getByTestId('sidebar')).toBeInTheDocument()
    })
    expect(screen.getByTestId('status-bar')).toBeInTheDocument()
    expect(screen.getByTestId('toast-container')).toBeInTheDocument()
  })

  it('renders skip-to-content link', async () => {
    await renderApp()

    expect(screen.getByText('Skip to main content')).toBeInTheDocument()
  })

  it('renders SetupWizard when showWizard is true', async () => {
    setStoreState({ showWizard: true })
    const { default: App } = await import('../App')

    await act(async () => {
      render(<App />)
    })

    await waitFor(() => {
      expect(screen.getByTestId('setup-wizard')).toBeInTheDocument()
    })
    expect(screen.queryByTestId('sidebar')).not.toBeInTheDocument()
    expect(screen.queryByTestId('scene-selector')).not.toBeInTheDocument()
  })

  it('renders WorkspaceOnboarding when showOnboarding is true', async () => {
    setStoreState({ showOnboarding: true, showWizard: false })
    const { default: App } = await import('../App')

    await act(async () => {
      render(<App />)
    })

    await waitFor(() => {
      expect(screen.getByTestId('workspace-onboarding')).toBeInTheDocument()
    })
    expect(screen.queryByTestId('sidebar')).not.toBeInTheDocument()
  })

  it('renders SceneSelector in main content area', async () => {
    setStoreState({ currentScene: 'terminal' })
    await renderApp()

    await waitFor(() => {
      expect(screen.getByTestId('scene-selector')).toBeInTheDocument()
    })
  })

  it('renders SlideOverPanel when slideOverContent is set and currentScene is not terminal', async () => {
    setStoreState({
      currentScene: 'agents-view',
      slideOverContent: { type: 'agent-form', title: 'Edit Agent', data: undefined },
    })
    await renderApp()

    expect(screen.getByTestId('slide-over-panel')).toBeInTheDocument()
    expect(screen.getByText('Edit Agent')).toBeInTheDocument()
  })

  it('does NOT render SlideOverPanel when currentScene is terminal', async () => {
    setStoreState({
      currentScene: 'terminal',
      slideOverContent: { type: 'agent-form', title: 'Edit Agent', data: undefined },
    })
    await renderApp()

    expect(screen.queryByTestId('slide-over-panel')).not.toBeInTheDocument()
  })

  it('does NOT render SlideOverPanel when slideOverContent is null', async () => {
    setStoreState({
      currentScene: 'agents-view',
      slideOverContent: null,
    })
    await renderApp()

    expect(screen.queryByTestId('slide-over-panel')).not.toBeInTheDocument()
  })

  it('closes slide-over when close button is clicked', async () => {
    setStoreState({
      currentScene: 'agents-view',
      slideOverContent: { type: 'agent-form', title: 'Edit Agent', data: undefined },
    })
    await renderApp()

    fireEvent.click(screen.getByTestId('slide-over-close'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith(null)
  })

  it('passes ollamaHealthy and nlpHealthy to StatusBar', async () => {
    setStoreState({ ollamaHealthy: true, nlpHealthy: false, projectID: 'proj-test' })
    await renderApp()

    const sb = screen.getByTestId('status-bar')
    expect(sb.getAttribute('data-ollama')).toBe('true')
    expect(sb.getAttribute('data-nlp')).toBe('false')
    expect(sb.getAttribute('data-projectid')).toBe('proj-test')
  })

  it('toggles command palette on Cmd+K', async () => {
    setStoreState({ isCommandPaletteOpen: false })
    await renderApp()

    await act(async () => {
      fireEvent.keyDown(window, { key: 'k', metaKey: true })
    })

    expect(mockSetIsCommandPaletteOpen).toHaveBeenCalled()
  })

  it('toggles command palette on Ctrl+K', async () => {
    setStoreState({ isCommandPaletteOpen: true })
    await renderApp()

    await act(async () => {
      fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
    })

    expect(mockSetIsCommandPaletteOpen).toHaveBeenCalled()
  })

  it('passes projectID to Sidebar', async () => {
    setStoreState({ projectID: 'my-project-id' })
    await renderApp()

    const sidebar = screen.getByTestId('sidebar')
    expect(sidebar.getAttribute('data-projectid')).toBe('my-project-id')
  })
})
