import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ToolsView } from '../ToolsView'

interface Tool {
  id: string
  name: string
  description: string
  code: string
}

// --- Mocks ---

const mockSetSlideOverContent = vi.fn()
const mockSetTools = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        const state = {
          setTools: mockSetTools,
          selectedObject: 'proj-1',
          setSlideOverContent: mockSetSlideOverContent,
          tools: [],
        }
        return selector(state)
      }
      return {
        setTools: mockSetTools,
        selectedObject: 'proj-1',
        setSlideOverContent: mockSetSlideOverContent,
        tools: [],
      }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({ setSlideOverContent: mockSetSlideOverContent })),
    },
  ),
}))

vi.mock('../../hooks/useCursorPagination', () => ({
  useCursorPagination: vi.fn(({ initialItems }: { initialItems: Tool[] }) => ({
    items: initialItems,
    hasMore: false,
    loadMore: vi.fn(),
    loading: false,
  })),
}))

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['', vi.fn()]),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'tools.title': 'Toolbox',
      'tools.subtitle': 'Strumenti modulari che estendono le capacità degli agenti.',
      'tools.create': 'Nuovo Tool',
      'tools.search': 'Cerca...',
      'generic.loadMore': 'Carica Altri',
      'generic.loadingLower': 'Caricamento...',
    }
    return map[key] ?? key
  },
}))

// --- Helpers ---

function makeTool(id: string, overrides?: Partial<Tool>): Tool {
  return {
    id,
    name: `Tool ${id}`,
    description: `Description ${id}`,
    code: `console.log("${id}")`,
    ...overrides,
  }
}

// --- Tests ---

describe('ToolsView', () => {
  const mockOnCreate = vi.fn()
  const mockOnEdit = vi.fn()
  const mockOnDelete = vi.fn()
  const mockOnExecute = vi.fn()

  beforeEach(() => { vi.clearAllMocks() })

  it('renders title and subtitle', () => {
    render(
      <ToolsView
        tools={[]}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    expect(screen.getByText('Toolbox')).toBeInTheDocument()
  })

  it('renders tool cards', () => {
    const tools = [makeTool('1'), makeTool('2')]
    render(
      <ToolsView
        tools={tools}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    expect(screen.getByText('Tool 1')).toBeInTheDocument()
    expect(screen.getByText('Tool 2')).toBeInTheDocument()
  })

  it('displays tool description', () => {
    const tools = [makeTool('1', { description: 'Does amazing things' })]
    render(
      <ToolsView
        tools={tools}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    expect(screen.getByText('Does amazing things')).toBeInTheDocument()
  })

  it('shows code preview', () => {
    const tools = [makeTool('1', { code: 'return 42' })]
    render(
      <ToolsView
        tools={tools}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    expect(screen.getByText('return 42')).toBeInTheDocument()
  })

  // — Empty state —

  it('shows empty state when no tools', () => {
    render(
      <ToolsView
        tools={[]}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    expect(screen.getByText(/Nessun strumento configurato/i)).toBeInTheDocument()
  })

  // — Loading / Error —

  it('renders skeleton when isLoading=true', () => {
    render(
      <ToolsView
        tools={[]}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
        isLoading={true}
      />,
    )
    const pulseEls = document.querySelectorAll('.animate-pulse')
    expect(pulseEls.length).toBeGreaterThan(0)
  })

  it('renders error when provided', () => {
    render(
      <ToolsView
        tools={[]}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
        error="Tool fetch error"
      />,
    )
    expect(screen.getByText('Tool fetch error')).toBeInTheDocument()
  })

  // — Interactions —

  it('opens slide over on create', () => {
    render(
      <ToolsView
        tools={[makeTool('1')]}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    fireEvent.click(screen.getByLabelText('Create new tool'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'tool-form',
      title: 'Nuovo Tool',
      data: undefined,
    })
  })

  it('calls onExecuteTool on execute button click', () => {
    const tools = [makeTool('1')]
    render(
      <ToolsView
        tools={tools}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    fireEvent.click(screen.getByLabelText('Execute tool Tool 1'))
    expect(mockOnExecute).toHaveBeenCalledWith('1')
  })

  it('calls onEditTool on details button click', () => {
    const tools = [makeTool('1')]
    render(
      <ToolsView
        tools={tools}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    const detailsBtn = screen.getByLabelText('View tool details')
    fireEvent.click(detailsBtn)
    const editBtn = screen.getByText('Edit')
    fireEvent.click(editBtn)
    expect(mockOnEdit).toHaveBeenCalledWith(tools[0])
  })

  // — Accessibility —

  it('has region role with aria-label', () => {
    render(
      <ToolsView
        tools={[]}
        onCreateTool={mockOnCreate}
        onEditTool={mockOnEdit}
        onDeleteTool={mockOnDelete}
        onExecuteTool={mockOnExecute}
      />,
    )
    expect(screen.getByRole('region', { name: 'Tools' })).toBeInTheDocument()
  })
})
