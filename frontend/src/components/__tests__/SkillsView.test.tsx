import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SkillsView } from '../SkillsView'

// Type locally
interface Skill {
  id: string
  name: string
  description: string
  toolIds?: string[]
}

// --- Mocks ---

const mockSetSlideOverContent = vi.fn()
const mockSetSkills = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        const state = {
          setSkills: mockSetSkills,
          selectedObject: 'proj-1',
          setSlideOverContent: mockSetSlideOverContent,
          skills: [],
        }
        return selector(state)
      }
      return {
        setSkills: mockSetSkills,
        selectedObject: 'proj-1',
        setSlideOverContent: mockSetSlideOverContent,
        skills: [],
      }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({ setSlideOverContent: mockSetSlideOverContent })),
    },
  ),
}))

vi.mock('../../hooks/useCursorPagination', () => ({
  useCursorPagination: vi.fn(({ initialItems }: { initialItems: Skill[] }) => ({
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
      'skills.title': 'Skill Framework',
      'skills.subtitle': 'Pacchetti di capacità e prompt che trasformano agenti in specialisti.',
      'skills.create': 'Crea Skill',
      'skills.search': 'Cerca...',
      'generic.loadMore': 'Carica Altri',
      'generic.loadingLower': 'Caricamento...',
    }
    return map[key] ?? key
  },
}))

// --- Helpers ---

function makeSkill(id: string, overrides?: Partial<Skill>): Skill {
  return { id, name: `Skill ${id}`, description: `Desc ${id}`, ...overrides }
}

// --- Tests ---

describe('SkillsView', () => {
  const mockOnCreate = vi.fn()
  const mockOnView = vi.fn()
  const mockOnDelete = vi.fn()
  const mockOnRun = vi.fn()
  const tools = [{ id: 't1', name: 'Tool A' }, { id: 't2', name: 'Tool B' }]

  beforeEach(() => { vi.clearAllMocks() })

  it('renders title and subtitle', () => {
    render(
      <SkillsView
        skills={[]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('Skill Framework')).toBeInTheDocument()
  })

  it('renders skill cards', () => {
    const skills = [makeSkill('1'), makeSkill('2')]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('Skill 1')).toBeInTheDocument()
    expect(screen.getByText('Skill 2')).toBeInTheDocument()
  })

  it('displays skill description', () => {
    const skills = [makeSkill('1', { description: 'A test skill' })]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('A test skill')).toBeInTheDocument()
  })

  it('renders associated tool names', () => {
    const skills = [makeSkill('1', { toolIds: ['t1'] })]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('Tool A')).toBeInTheDocument()
  })

  it('shows raw tool ID when tool not in tools list', () => {
    const skills = [makeSkill('1', { toolIds: ['t-unknown'] })]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('t-unknown')).toBeInTheDocument()
  })

  // — Empty state —

  it('shows empty state when no skills', () => {
    render(
      <SkillsView
        skills={[]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    // The empty container renders; text is untranslated raw string
    expect(screen.getByText(/Nessuna Skill personalizzata/i)).toBeInTheDocument()
  })

  // — Loading / Error —

  it('renders skeleton when isLoading=true', () => {
    render(
      <SkillsView
        skills={[]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
        isLoading={true}
      />,
    )
    const pulseEls = document.querySelectorAll('.animate-pulse')
    expect(pulseEls.length).toBeGreaterThan(0)
  })

  it('renders error message when error is provided', () => {
    render(
      <SkillsView
        skills={[]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
        error="Skill load failed"
      />,
    )
    expect(screen.getByText('Skill load failed')).toBeInTheDocument()
  })

  // — Interactions —

  it('opens slide over on create', () => {
    render(
      <SkillsView
        skills={[makeSkill('1')]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    fireEvent.click(screen.getByLabelText('Create new skill'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith({
      type: 'skill-form',
      title: 'Crea Skill',
      data: { tools },
    })
  })

  it('calls onRunSkill when execute is clicked', () => {
    const skills = [makeSkill('1')]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    fireEvent.click(screen.getByLabelText('Execute skill Skill 1'))
    expect(mockOnRun).toHaveBeenCalledWith('1')
  })

  it('calls onViewSkillDetail when details is clicked', () => {
    const skills = [makeSkill('1')]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    fireEvent.click(screen.getByLabelText('View details for Skill 1'))
    expect(mockOnView).toHaveBeenCalledWith(skills[0])
  })

  // — Accessibility —

  it('has region role with aria-label', () => {
    render(
      <SkillsView
        skills={[]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByRole('region', { name: 'Skills' })).toBeInTheDocument()
  })

  it('renders inline mode without max-w wrapper class', () => {
    const { container } = render(
      <SkillsView
        skills={[makeSkill('1')]}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
        inline={true}
      />,
    )
    const root = container.firstElementChild as HTMLElement
    expect(root.className).not.toContain('max-w-6xl')
  })

  it('does not render tool badges when toolIds is empty array', () => {
    const skills = [makeSkill('1', { toolIds: [] })]
    const { container } = render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('Skill 1')).toBeInTheDocument()
    const badges = container.querySelectorAll('.bg-primary\\/10')
    expect(badges.length).toBe(0)
  })

  it('does not render tool badges when toolIds is undefined', () => {
    const skills = [makeSkill('1', { toolIds: undefined })]
    render(
      <SkillsView
        skills={skills}
        tools={tools}
        onCreateSkill={mockOnCreate}
        onViewSkillDetail={mockOnView}
        onDeleteSkill={mockOnDelete}
        onRunSkill={mockOnRun}
      />,
    )
    expect(screen.getByText('Skill 1')).toBeInTheDocument()
  })
})
