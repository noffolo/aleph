import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { SkillForm } from '../SkillForm'

vi.mock('@/store/useStore', () => ({ useStore: vi.fn(() => ({})) }))

describe('SkillForm', () => {
  const mockOnSave = vi.fn()
  const mockOnCancel = vi.fn()
  const defaultProps = { tools: [], onSave: mockOnSave, onCancel: mockOnCancel, title: 'Test Skill Form' }

  beforeEach(() => { vi.clearAllMocks() })

  it('renders title', () => {
    render(<SkillForm {...defaultProps} />)
    expect(screen.getByText('Test Skill Form')).toBeInTheDocument()
  })

  it('has submit and cancel buttons', () => {
    render(<SkillForm {...defaultProps} />)
    expect(screen.getByText(/Crea Skill/i)).toBeInTheDocument()
    expect(screen.getByText(/Annulla/i)).toBeInTheDocument()
  })

  it('calls onCancel on cancel click', () => {
    render(<SkillForm {...defaultProps} />)
    fireEvent.click(screen.getByText(/Annulla/i))
    expect(mockOnCancel).toHaveBeenCalled()
  })
})
