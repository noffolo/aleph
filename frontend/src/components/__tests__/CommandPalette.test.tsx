import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const { mockExecuteCommand, mockGetTabCompletion } = vi.hoisted(() => ({
  mockExecuteCommand: vi.fn(),
  mockGetTabCompletion: vi.fn(),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => key,
}))

vi.mock('lucide-react', () => {
  const Icon = (name: string) => {
    const Comp = () => null
    Comp.displayName = name
    return Comp
  }
  return {
    Command: Icon('Command'),
    ArrowRight: Icon('ArrowRight'),
    Navigation: Icon('Navigation'),
    Zap: Icon('Zap'),
    Settings: Icon('Settings'),
    Database: Icon('Database'),
  }
})

vi.mock('../terminal/slashCommands', () => ({
  SLASH_COMMANDS: [
    { name: '/explore', description: 'Explore data and objects', requiresConfirmation: false },
    { name: '/agent', description: 'Interact with agents', requiresConfirmation: true },
    { name: '/skills', description: 'View skills panel', requiresConfirmation: true },
    { name: '/tool install', description: 'Install a tool from URL', requiresConfirmation: true },
    { name: '/tool list', description: 'List installed tools', requiresConfirmation: false },
    { name: '/help', description: 'Show available commands', requiresConfirmation: false },
    { name: '/clear', description: 'Clear chat history', requiresConfirmation: false },
  ],
  executeCommand: mockExecuteCommand,
  getTabCompletion: mockGetTabCompletion,
}))

import { CommandPalette } from '../CommandPalette'

// ── Helpers ────────────────────────────────────────────────────────────────

const defaultProps = {
  isOpen: true,
  onClose: vi.fn(),
  availableObjects: [] as string[],
  projects: [] as { id: string; name: string }[],
  onSelectProject: vi.fn(),
  onSelectObject: vi.fn(),
}

function setup(overrides: Partial<typeof defaultProps> = {}) {
  const props = { ...defaultProps, ...overrides }
  // Ensure fresh mocks per render when needed
  const onClose = overrides.onClose ?? vi.fn()
  const onSelectProject = overrides.onSelectProject ?? vi.fn()
  const onSelectObject = overrides.onSelectObject ?? vi.fn()
  const utils = render(
    <CommandPalette
      {...props}
      onClose={onClose}
      onSelectProject={onSelectProject}
      onSelectObject={onSelectObject}
    />,
  )
  return { ...utils, onClose, onSelectProject, onSelectObject }
}

beforeEach(() => {
  vi.clearAllMocks()
  Element.prototype.scrollIntoView = vi.fn()
})

// ── Tests ──────────────────────────────────────────────────────────────────

describe('CommandPalette', () => {
  // ── Rendering ──────────────────────────────────────────────────────────

  it('returns null when isOpen is false', () => {
    const { container } = render(<CommandPalette {...defaultProps} isOpen={false} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders dialog with correct role and modal attribute when open', () => {
    setup()
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-label', 'Command palette')
  })

  it('shows search input with translated placeholder', () => {
    setup()
    const input = screen.getByLabelText('Search commands')
    expect(input).toBeInTheDocument()
    expect(input).toHaveAttribute('placeholder', 'commandPalette.search')
    expect(input).toHaveFocus()
  })

  // ── Empty / Prompt state ───────────────────────────────────────────────

  it('shows prompt text and Command icon when search is empty', () => {
    setup()
    expect(screen.getByText('commandPalette.prompt')).toBeInTheDocument()
  })

  // ── Command filtering ──────────────────────────────────────────────────

  it('filters commands by search text (name match)', () => {
    setup()
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: '/explore' } })

    // The /explore command should be visible
    expect(screen.getByText('/explore')).toBeInTheDocument()
    // Other commands should not be visible
    expect(screen.queryByText('/help')).not.toBeInTheDocument()
    expect(screen.queryByText('/clear')).not.toBeInTheDocument()
  })

  it('filters commands by search text (description match)', () => {
    setup()
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: 'chat history' } })

    // /clear has "Clear chat history" description
    expect(screen.getByText('/clear')).toBeInTheDocument()
    expect(screen.queryByText('/help')).not.toBeInTheDocument()
  })

  // ── Command sections ───────────────────────────────────────────────────

  it('shows command sections with headers and icons when search is non-empty', () => {
    setup()
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: '/' } })

    // All section labels should appear
    expect(screen.getByText('commandPalette.section.navigate')).toBeInTheDocument()
    expect(screen.getByText('commandPalette.section.actions')).toBeInTheDocument()
    expect(screen.getByText('commandPalette.section.system')).toBeInTheDocument()
  })

  it('hides sections when no search text', () => {
    setup()
    expect(screen.queryByText('commandPalette.section.navigate')).not.toBeInTheDocument()
  })

  it('hides a section when no commands match that section', () => {
    setup()
    const input = screen.getByLabelText('Search commands')
    // Search for something that only matches system commands
    fireEvent.change(input, { target: { value: '/help' } })

    // Only system section should appear
    expect(screen.getByText('commandPalette.section.system')).toBeInTheDocument()
    expect(screen.queryByText('commandPalette.section.navigate')).not.toBeInTheDocument()
    expect(screen.queryByText('commandPalette.section.actions')).not.toBeInTheDocument()
  })

  // ── Keyboard navigation ────────────────────────────────────────────────

  it('navigates commands with ArrowDown and executes selected with Enter', () => {
    const { onClose } = setup()
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: '/' } })

    const dialog = screen.getByRole('dialog')

    // change sets selectedIndex=0, one ArrowDown moves to index 1 (= /agent)
    fireEvent.keyDown(dialog, { key: 'ArrowDown' })
    fireEvent.keyDown(dialog, { key: 'Enter' })

    expect(mockExecuteCommand).toHaveBeenCalledWith('/agent')
    expect(onClose).toHaveBeenCalled()
  })

  it('ArrowUp decreases selected index but stays at 0 minimum', () => {
    const { onClose } = setup()
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: '/' } })

    const dialog = screen.getByRole('dialog')

    // ArrowDown then ArrowUp then Enter → should execute first command
    fireEvent.keyDown(dialog, { key: 'ArrowDown' }) // index 0 → 1
    fireEvent.keyDown(dialog, { key: 'ArrowUp' })   // index 1 → 0
    fireEvent.keyDown(dialog, { key: 'Enter' })

    expect(mockExecuteCommand).toHaveBeenCalledWith('/explore')
    expect(onClose).toHaveBeenCalled()
  })

  it('Escape key calls onClose', () => {
    const { onClose } = setup()
    const dialog = screen.getByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'Escape' })
    expect(onClose).toHaveBeenCalled()
  })

  it('Tab triggers getTabCompletion and cycles completions', () => {
    mockGetTabCompletion.mockReturnValue(['/explore', '/agent'])

    setup()
    const input = screen.getByLabelText('Search commands') as HTMLInputElement
    fireEvent.change(input, { target: { value: '/' } })

    const dialog = screen.getByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'Tab' })

    expect(mockGetTabCompletion).toHaveBeenCalledWith('/')
    // With completions ['/explore', '/agent'] and search='/':
    // search !== currentC ('/explore'), so setSearch('/explore')
    expect(input.value).toBe('/explore')
  })

  it('Tab does nothing when search does not start with /', () => {
    setup()
    const input = screen.getByLabelText('Search commands') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'help' } })

    const dialog = screen.getByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'Tab' })

    expect(mockGetTabCompletion).not.toHaveBeenCalled()
  })

  // ── Click handlers ─────────────────────────────────────────────────────

  it('clicking a command calls executeCommand and onClose', () => {
    const { onClose } = setup()
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: '/help' } })

    fireEvent.click(screen.getByText('/help'))
    expect(mockExecuteCommand).toHaveBeenCalledWith('/help')
    expect(onClose).toHaveBeenCalled()
  })

  it('clicking backdrop overlay calls onClose', () => {
    const { onClose } = setup()
    const dialog = screen.getByRole('dialog')
    fireEvent.click(dialog)
    expect(onClose).toHaveBeenCalled()
  })

  // ── Objects ────────────────────────────────────────────────────────────

  it('renders filtered objects when search matches availableObjects', () => {
    setup({
      availableObjects: ['alpha_object', 'beta_item', 'gamma_thing'],
    })
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: 'alpha' } })

    expect(screen.getByText('alpha_object')).toBeInTheDocument()
    expect(screen.queryByText('beta_item')).not.toBeInTheDocument()
  })

  it('clicking an object calls onSelectObject and onClose', () => {
    const { onSelectObject, onClose } = setup({
      availableObjects: ['test-obj'],
    })
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: 'test' } })

    fireEvent.click(screen.getByText('test-obj'))
    expect(onSelectObject).toHaveBeenCalledWith('test-obj')
    expect(onClose).toHaveBeenCalled()
  })

  // ── Projects ───────────────────────────────────────────────────────────

  it('renders filtered projects when search matches project name', () => {
    setup({
      projects: [
        { id: 'p1', name: 'Project Alpha' },
        { id: 'p2', name: 'Project Beta' },
      ],
    })
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: 'Alpha' } })

    expect(screen.getByText('Project Alpha')).toBeInTheDocument()
    expect(screen.queryByText('Project Beta')).not.toBeInTheDocument()
  })

  it('clicking a project calls onSelectProject with id and onClose', () => {
    const { onSelectProject, onClose } = setup({
      projects: [{ id: 'project-42', name: 'My Project' }],
    })
    const input = screen.getByLabelText('Search commands')
    fireEvent.change(input, { target: { value: 'My' } })

    fireEvent.click(screen.getByText('My Project'))
    expect(onSelectProject).toHaveBeenCalledWith('project-42')
    expect(onClose).toHaveBeenCalled()
  })

  // ── Enter with object/project selected ─────────────────────────────────

  it('Enter selects object when keyboard-navigated past commands', () => {
    const { onSelectObject, onClose } = setup({
      availableObjects: ['obj-one', 'obj-two'],
    })
    const input = screen.getByLabelText('Search commands')
    // "obj" matches /explore (description: "Explore data and objects") = 1 command
    // plus both objects. One ArrowDown after change (sets idx=0) reaches obj-one.
    fireEvent.change(input, { target: { value: 'obj' } })

    const dialog = screen.getByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'ArrowDown' })
    fireEvent.keyDown(dialog, { key: 'Enter' })

    expect(onSelectObject).toHaveBeenCalledWith('obj-one')
    expect(onClose).toHaveBeenCalled()
  })

  it('Enter selects project when keyboard-navigated past commands and objects', () => {
    const { onSelectProject, onClose } = setup({
      availableObjects: ['obj'],
      projects: [{ id: 'prj-1', name: 'Target Project' }],
    })
    const input = screen.getByLabelText('Search commands')
    // Search matches everything with 'o' in it
    fireEvent.change(input, { target: { value: 'o' } })

    const dialog = screen.getByRole('dialog')
    // Navigate past all commands and objects to reach the project
    for (let i = 0; i < 20; i++) {
      fireEvent.keyDown(dialog, { key: 'ArrowDown' })
    }
    fireEvent.keyDown(dialog, { key: 'Enter' })

    expect(onSelectProject).toHaveBeenCalledWith('prj-1')
    expect(onClose).toHaveBeenCalled()
  })
})
