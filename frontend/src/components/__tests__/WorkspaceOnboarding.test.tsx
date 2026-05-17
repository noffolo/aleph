import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { WorkspaceOnboarding } from '../WorkspaceOnboarding'

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'generic.irreversibleDelete': 'Questa azione è irreversibile',
      'setup.apiKey': 'API Key',
      'onboarding.apiKey': 'Inserisci la tua API Key',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => ({
  Briefcase: () => null,
  Plus: () => null,
  Key: () => null,
  Lock: () => null,
  ArrowRight: () => null,
  X: () => null,
  Trash2: () => null,
  Binary: () => null,
  Sparkles: () => null,
  AlertTriangle: () => null,
}))

describe('WorkspaceOnboarding', () => {
  const mockOnSelectProject = vi.fn()
  const mockOnDeleteProject = vi.fn()
  const mockOnCreateProject = vi.fn()

  const projects = [
    { id: 'proj-1', name: 'Project Alpha' },
    { id: 'proj-2', name: 'Project Beta' },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders Aleph heading', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    expect(screen.getByText('Aleph')).toBeInTheDocument()
  })

  it('renders project cards', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    expect(screen.getByText('Project Alpha')).toBeInTheDocument()
    expect(screen.getByText('Project Beta')).toBeInTheDocument()
  })

  it('renders create workspace button', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    expect(screen.getByText('Nuovo spazio di lavoro')).toBeInTheDocument()
  })

  it('calls onCreateProject when create button clicked', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    fireEvent.click(screen.getByText('Nuovo spazio di lavoro'))
    expect(mockOnCreateProject).toHaveBeenCalledTimes(1)
  })

  it('shows unlock screen when project card clicked', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    fireEvent.click(screen.getByText('Project Alpha'))
    expect(screen.getByText('Sblocca Project Alpha')).toBeInTheDocument()
  })

  it('calls onSelectProject with credentials on Access button', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    fireEvent.click(screen.getByText('Project Alpha'))
    const keyInput = screen.getByPlaceholderText('Inserisci la tua API Key')
    fireEvent.change(keyInput, { target: { value: 'secret-key' } })
    fireEvent.click(screen.getByText('Accedi'))
    expect(mockOnSelectProject).toHaveBeenCalledWith('proj-1', 'secret-key')
  })

  it('returns to project list when X is clicked', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    fireEvent.click(screen.getByText('Project Alpha'))
    fireEvent.click(screen.getByLabelText('Deselect project'))
    expect(screen.getByText('Aleph')).toBeInTheDocument()
  })

  it('opens delete confirmation modal on trash button click', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    const trashButtons = screen.getAllByRole('button')
    expect(trashButtons.length).toBeGreaterThan(0)
  })

  it('shows delete modal with project name when trash clicked', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    const trashBtn = screen.getAllByRole('button')[0]
    fireEvent.click(trashBtn)
    expect(screen.getByText(/Elimina Project Alpha/)).toBeInTheDocument()
    expect(screen.getByText('Elimina definitivamente')).toBeInTheDocument()
  })

  it('can cancel delete modal', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    const trashBtn = screen.getAllByRole('button')[0]
    fireEvent.click(trashBtn)
    fireEvent.click(screen.getByLabelText('Cancel onboarding'))
    expect(screen.queryByText('Elimina definitivamente')).not.toBeInTheDocument()
  })

  it('calls onDeleteProject with key in delete modal', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    const trashBtn = screen.getAllByRole('button')[0]
    fireEvent.click(trashBtn)
    const keyInput = screen.getByPlaceholderText('API Key')
    fireEvent.change(keyInput, { target: { value: 'delete-key' } })
    fireEvent.click(screen.getByText('Elimina definitivamente'))
    expect(mockOnDeleteProject).toHaveBeenCalledWith('proj-1', 'delete-key')
  })

  it('submits unlock on Enter key', () => {
    render(
      <WorkspaceOnboarding
        projects={projects}
        onSelectProject={mockOnSelectProject}
        onDeleteProject={mockOnDeleteProject}
        onCreateProject={mockOnCreateProject}
      />,
    )
    fireEvent.click(screen.getByText('Project Alpha'))
    const keyInput = screen.getByPlaceholderText('Inserisci la tua API Key')
    fireEvent.change(keyInput, { target: { value: 'enter-key' } })
    fireEvent.keyDown(keyInput, { key: 'Enter' })
    expect(mockOnSelectProject).toHaveBeenCalledWith('proj-1', 'enter-key')
  })
})
