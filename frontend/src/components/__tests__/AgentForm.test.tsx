import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { AgentForm } from '../AgentForm'

vi.mock('@/store/useStore', () => ({
  useStore: vi.fn(() => ({})),
}))

describe('AgentForm', () => {
  const mockOnSave = vi.fn()
  const mockOnCancel = vi.fn()
  const defaultProps = { onSave: mockOnSave, onCancel: mockOnCancel, title: 'Test Form' }

  beforeEach(() => { vi.clearAllMocks() })

  it('renders title', () => {
    render(<AgentForm {...defaultProps} />)
    expect(screen.getByText('Test Form')).toBeInTheDocument()
  })

  it('renders provider name field', () => {
    render(<AgentForm {...defaultProps} />)
    expect(screen.getByText('Provider')).toBeInTheDocument()
  })

  it('calls onCancel on cancel click', () => {
    render(<AgentForm {...defaultProps} />)
    fireEvent.click(screen.getByText(/Annulla/i))
    expect(mockOnCancel).toHaveBeenCalled()
  })

  it('has a name input with placeholder', () => {
    render(<AgentForm {...defaultProps} />)
    expect(screen.getByPlaceholderText(/Analista/i)).toBeInTheDocument()
  })
})
