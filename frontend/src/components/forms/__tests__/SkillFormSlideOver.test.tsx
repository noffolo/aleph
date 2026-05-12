import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SkillFormSlideOver } from '../SkillFormSlideOver'

vi.mock('../../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn(() => ({})),
    { getState: vi.fn(() => ({ setSlideOverContent: vi.fn() })) },
  ),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useSkillActions', () => ({
  useSkillActions: () => ({
    onCreateSkill: vi.fn(),
    onUpdateSkill: vi.fn(),
  }),
}))

vi.mock('../../../schemas', () => ({
  SkillSchema: { safeParse: vi.fn(() => ({ success: true, data: {} })) },
}))

vi.mock('../../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'skills.create': 'Crea Skill',
      'skills.edit': 'Modifica Skill',
      'skills.form.name': 'Nome skill',
      'skills.form.description': 'Descrizione',
      'confirmDialog.cancel': 'Annulla',
    }
    return map[key] ?? key
  },
}))

describe('SkillFormSlideOver', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders create title when no skill provided', () => {
    render(<SkillFormSlideOver tools={[]} />)
    expect(screen.getByRole('heading', { name: 'Crea Skill' })).toBeInTheDocument()
  })

  it('renders edit title when skill provided', () => {
    const skill = { id: 's1', name: 'Test Skill', description: 'desc', toolIds: [] }
    render(<SkillFormSlideOver skill={skill} tools={[]} />)
    expect(screen.getByRole('heading', { name: 'Modifica Skill' })).toBeInTheDocument()
  })

  it('renders name and description fields', () => {
    render(<SkillFormSlideOver tools={[]} />)
    expect(screen.getByLabelText('Nome')).toBeInTheDocument()
    expect(screen.getByLabelText('Descrizione')).toBeInTheDocument()
  })

  it('renders tools checkboxes', () => {
    const tools = [
      { id: 't1', name: 'Tool 1', description: '', code: '' },
      { id: 't2', name: 'Tool 2', description: '', code: '' },
    ]
    render(<SkillFormSlideOver tools={tools} />)
    expect(screen.getByText('Tool 1')).toBeInTheDocument()
    expect(screen.getByText('Tool 2')).toBeInTheDocument()
  })

  it('pre-fills fields in edit mode', () => {
    const skill = { id: 's1', name: 'MySkill', description: 'A skill', toolIds: [] }
    render(<SkillFormSlideOver skill={skill} tools={[]} />)
    expect(screen.getByDisplayValue('MySkill')).toBeInTheDocument()
    expect(screen.getByDisplayValue('A skill')).toBeInTheDocument()
  })

  it('renders cancel and submit buttons', () => {
    render(<SkillFormSlideOver tools={[]} />)
    expect(screen.getByText('Annulla')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Crea Skill' })).toBeInTheDocument()
  })

  it('checks tool checkbox on click', () => {
    const tools = [{ id: 't1', name: 'Tool 1', description: '', code: '' }]
    render(<SkillFormSlideOver tools={tools} />)
    const checkbox = screen.getByRole('checkbox')
    fireEvent.click(checkbox)
    expect(checkbox).toBeChecked()
  })
})
