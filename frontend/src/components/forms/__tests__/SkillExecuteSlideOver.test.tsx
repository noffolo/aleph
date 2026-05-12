import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SkillExecuteSlideOver } from '../SkillExecuteSlideOver'

vi.mock('../../../store/useStore', () => ({
  useStore: vi.fn((selector?: (state: unknown) => unknown) => {
    const state: Record<string, unknown> = {
      tools: [{ id: 't1', name: 'Tool A' }, { id: 't2', name: 'Tool B' }],
      sandboxInput: '{}',
      setSandboxInput: mockSetSandboxInput,
    }
    if (typeof selector === 'function') return selector(state)
    return state
  }),
}))

vi.mock('../../../hooks/useAppActions', () => ({
  useAppActions: () => ({ loadProjectData: vi.fn() }),
}))

vi.mock('../../../hooks/domain/useSkillActions', () => ({
  useSkillActions: () => ({ onRunSkill: vi.fn() }),
}))

const mockSetSandboxInput = vi.fn()

describe('SkillExecuteSlideOver', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const skill = { id: 's1', name: 'DataAnalyzer', description: 'Analyzes data', toolIds: ['t1', 't2'] }

  it('renders skill name as title', () => {
    render(<SkillExecuteSlideOver skill={skill} />)
    expect(screen.getByText('DataAnalyzer')).toBeInTheDocument()
  })

  it('renders skill description', () => {
    render(<SkillExecuteSlideOver skill={skill} />)
    expect(screen.getByText('Analyzes data')).toBeInTheDocument()
  })

  it('renders associated tools', () => {
    render(<SkillExecuteSlideOver skill={skill} />)
    expect(screen.getByText('Tool A')).toBeInTheDocument()
    expect(screen.getByText('Tool B')).toBeInTheDocument()
  })

  it('renders JSON input textarea', () => {
    render(<SkillExecuteSlideOver skill={skill} />)
    expect(screen.getByDisplayValue('{}')).toBeInTheDocument()
  })

  it('calls setSandboxInput on textarea change', () => {
    render(<SkillExecuteSlideOver skill={skill} />)
    const textarea = screen.getByDisplayValue('{}')
    fireEvent.change(textarea, { target: { value: '{"param": 1}' } })
    expect(mockSetSandboxInput).toHaveBeenCalledWith('{"param": 1}')
  })

  it('renders execute button', () => {
    render(<SkillExecuteSlideOver skill={skill} />)
    expect(screen.getByText('Esegui Skill nel Sandbox')).toBeInTheDocument()
  })

  it('returns null for skill without id', () => {
    const { container } = render(<SkillExecuteSlideOver skill={{ id: '', name: '', description: '' }} />)
    expect(container.firstChild).toBeNull()
  })
})
