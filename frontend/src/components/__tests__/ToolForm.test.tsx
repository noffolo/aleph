import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ToolForm } from '../ToolForm'

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(() => ({})), { getState: vi.fn(() => ({ apiKey: 'test' })) }),
}))

global.fetch = vi.fn()

describe('ToolForm', () => {
  const mockOnSave = vi.fn()
  const mockOnCancel = vi.fn()
  const defaultProps = { onSave: mockOnSave, onCancel: mockOnCancel, title: 'Test Tool Form' }

  beforeEach(() => { vi.clearAllMocks() })

  it('renders title', () => {
    render(<ToolForm {...defaultProps} />)
    expect(screen.getByText('Test Tool Form')).toBeInTheDocument()
  })

  it('has submit and cancel buttons', () => {
    render(<ToolForm {...defaultProps} />)
    expect(screen.getByText(/Crea Tool/i)).toBeInTheDocument()
    expect(screen.getByText(/Annulla/i)).toBeInTheDocument()
  })

  it('calls onCancel on cancel click', () => {
    render(<ToolForm {...defaultProps} />)
    fireEvent.click(screen.getByText(/Annulla/i))
    expect(mockOnCancel).toHaveBeenCalled()
  })

  it('submits form on valid data and calls fetch', async () => {
    (global.fetch as any).mockResolvedValueOnce({ ok: true, json: async () => ({}) })
    render(<ToolForm {...defaultProps} />)
    const input = screen.getByPlaceholderText(/Analizzatore/i)
    fireEvent.change(input, { target: { value: 'My Tool' } })
    fireEvent.click(screen.getByText(/Crea Tool/i))
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled()
    }, { timeout: 3000 })
  })
})
