import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

const mockSetSlideOverContent = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn(() => ({})),
    {
      getState: () => ({
        setSlideOverContent: mockSetSlideOverContent,
      }),
    },
  ),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'datasources.title': 'Sorgenti Dati',
      'datasources.subtitle': 'Sottotitolo',
      'datasources.create': 'Crea',
      'datasources.status.running': 'In Esecuzione',
      'datasources.status.completed': 'Completato',
      'datasources.status.failed': 'Fallito',
      'datasources.status.execute': 'Esegui',
      'datasources.noPipeline': 'Nessuna Pipeline',
      'datasources.empty': 'Crea la tua prima pipeline',
      'datasources.logOutput': 'Output Log',
      'datasources.confirmDelete': 'Sei sicuro?',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => {
  const Icon = (name: string) => {
    const Comp = (props: React.SVGProps<SVGSVGElement>) => <svg {...props} data-testid={`icon-${name}`} />
    Comp.displayName = name
    return Comp
  }
  return {
    Plus: Icon('Plus'),
    Activity: Icon('Activity'),
    X: Icon('X'),
    Trash2: Icon('Trash2'),
    Database: Icon('Database'),
    Globe: Icon('Globe'),
    FileText: Icon('FileText'),
    Link: Icon('Link'),
    Play: Icon('Play'),
    Mail: Icon('Mail'),
    Rss: Icon('Rss'),
    Github: Icon('Github'),
    Terminal: Icon('Terminal'),
  }
})

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
}))

vi.mock('../ui/InlineError', () => ({
  InlineError: ({ message }: { message: string }) => <div data-testid="inline-error">{message}</div>,
}))

import { DataSourcesView } from '../DataSourcesView'

const makeTask = (overrides?: Record<string, unknown>) => ({
  id: 'task-1',
  name: 'Import CSV',
  sourceType: 'csv',
  status: 'idle',
  progress: 0,
  ...overrides,
})

const defaultProps = {
  tasks: [] as ReturnType<typeof makeTask>[],
  onAddSource: vi.fn(),
  onRunTask: vi.fn(),
  onViewLogs: vi.fn(),
  onDeleteTask: vi.fn(),
  taskLogs: '',
  setTaskLogs: vi.fn(),
}

describe('DataSourcesView', () => {
  let props: typeof defaultProps

  beforeEach(() => {
    vi.clearAllMocks()
    props = {
      ...defaultProps,
      onAddSource: vi.fn(),
      onRunTask: vi.fn(),
      onViewLogs: vi.fn(),
      onDeleteTask: vi.fn(),
      setTaskLogs: vi.fn(),
    }
  })

  // --- Render states ---
  it('renders skeleton when isLoading is true', () => {
    render(<DataSourcesView {...props} isLoading={true} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error when error is provided', () => {
    render(<DataSourcesView {...props} error="Connection failed" />)
    expect(screen.getByTestId('inline-error')).toHaveTextContent('Connection failed')
  })

  it('renders title and subtitle', () => {
    render(<DataSourcesView {...props} />)
    expect(screen.getByText('Sorgenti Dati')).toBeInTheDocument()
    expect(screen.getByText('Sottotitolo')).toBeInTheDocument()
  })

  // --- Empty state ---
  it('renders empty state when no tasks', () => {
    render(<DataSourcesView {...props} />)
    expect(screen.getByText('Nessuna Pipeline')).toBeInTheDocument()
    expect(screen.getByText('Crea la tua prima pipeline')).toBeInTheDocument()
  })

  it('opens datasource form from empty state button', () => {
    render(<DataSourcesView {...props} />)
    fireEvent.click(screen.getAllByText('Crea')[0])
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'datasource-form',
      title: 'Sorgenti Dati',
      data: undefined,
    })
  })

  it('opens datasource form from header button', () => {
    render(<DataSourcesView {...props} />)
    const buttons = screen.getAllByLabelText('Add data source')
    fireEvent.click(buttons[0]) // First button is the header one
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'datasource-form',
      title: 'Sorgenti Dati',
      data: undefined,
    })
  })

  // --- Task rendering ---
  it('renders task name and source type', () => {
    const tasks = [makeTask()]
    render(<DataSourcesView {...props} tasks={tasks} />)
    expect(screen.getByText('Import CSV')).toBeInTheDocument()
    expect(screen.getByText('csv')).toBeInTheDocument()
  })

  it('renders progress bar with percentage', () => {
    const tasks = [makeTask({ progress: 75 })]
    render(<DataSourcesView {...props} tasks={tasks} />)
    expect(screen.getByText('75%')).toBeInTheDocument()
  })

  // --- Status-based UI ---
  it('shows running status text for running tasks', () => {
    const tasks = [makeTask({ status: 'esecuzione' })]
    render(<DataSourcesView {...props} tasks={tasks} />)
    expect(screen.getByText('In Esecuzione')).toBeInTheDocument()
  })

  it('shows completed status text for completed tasks', () => {
    const tasks = [makeTask({ status: 'completato' })]
    render(<DataSourcesView {...props} tasks={tasks} />)
    expect(screen.getByText('Completato')).toBeInTheDocument()
  })

  it('shows failed status text for failed tasks', () => {
    const tasks = [makeTask({ status: 'fallito' })]
    render(<DataSourcesView {...props} tasks={tasks} />)
    expect(screen.getByText('Fallito')).toBeInTheDocument()
  })

  it('shows execute button text for idle tasks', () => {
    const tasks = [makeTask({ status: 'idle' })]
    render(<DataSourcesView {...props} tasks={tasks} />)
    expect(screen.getByText('Esegui')).toBeInTheDocument()
  })

  // --- Button interactions ---
  it('calls onViewLogs with task id on Logs click', () => {
    const tasks = [makeTask()]
    render(<DataSourcesView {...props} tasks={tasks} />)
    fireEvent.click(screen.getByText('Logs'))
    expect(props.onViewLogs).toHaveBeenCalledWith('task-1')
  })

  it('calls onRunTask with task id on play click', () => {
    const tasks = [makeTask()]
    render(<DataSourcesView {...props} tasks={tasks} />)
    fireEvent.click(screen.getByText('Esegui'))
    expect(props.onRunTask).toHaveBeenCalledWith('task-1')
  })

  it('disables run button when task is running', () => {
    const tasks = [makeTask({ status: 'running' })]
    render(<DataSourcesView {...props} tasks={tasks} />)
    const runBtn = screen.getByText('In Esecuzione')
    expect(runBtn.closest('button')).toBeDisabled()
  })

  it('calls onDeleteTask after confirmation', () => {
    window.confirm = vi.fn(() => true)
    const tasks = [makeTask()]
    render(<DataSourcesView {...props} tasks={tasks} />)
    fireEvent.click(screen.getByLabelText('Delete task Import CSV'))
    expect(window.confirm).toHaveBeenCalledWith('Sei sicuro?')
    expect(props.onDeleteTask).toHaveBeenCalledWith('task-1')
  })

  // --- Log panel ---
  it('renders log output when taskLogs is non-empty', () => {
    render(<DataSourcesView {...props} taskLogs="some log output" />)
    expect(screen.getByText('Output Log')).toBeInTheDocument()
    expect(screen.getByText('some log output')).toBeInTheDocument()
  })

  it('closes log panel on X button click', () => {
    render(<DataSourcesView {...props} taskLogs="log data" />)
    fireEvent.click(screen.getByLabelText('Close log panel'))
    expect(props.setTaskLogs).toHaveBeenCalledWith('')
  })

  it('does not render log panel when taskLogs is empty', () => {
    render(<DataSourcesView {...props} taskLogs="" />)
    expect(screen.queryByText('Output Log')).not.toBeInTheDocument()
  })

  // --- Inline mode ---
  it('applies inline styling when inline prop is true', () => {
    const tasks = [makeTask()]
    const { container } = render(<DataSourcesView {...props} tasks={tasks} inline={true} />)
    const root = container.firstElementChild as HTMLElement
    expect(root.className).not.toContain('max-w-6xl')
  })
})
