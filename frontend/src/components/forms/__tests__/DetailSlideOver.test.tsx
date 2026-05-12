import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DetailSlideOver } from '../DetailSlideOver'

describe('DetailSlideOver', () => {
  const mockOnClose = vi.fn()

  it('renders title', () => {
    render(<DetailSlideOver data={{}} onClose={mockOnClose} title="Test Detail" />)
    expect(screen.getByText('Test Detail')).toBeInTheDocument()
  })

  it('renders default title when none provided', () => {
    render(<DetailSlideOver data={{}} onClose={mockOnClose} />)
    expect(screen.getByText('Dettaglio')).toBeInTheDocument()
  })

  it('renders all data key-value pairs', () => {
    const data = { name: 'Alice', role: 'Admin', active: 'true' }
    render(<DetailSlideOver data={data} onClose={mockOnClose} />)
    expect(screen.getByText('name')).toBeInTheDocument()
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('role')).toBeInTheDocument()
    expect(screen.getByText('Admin')).toBeInTheDocument()
  })

  it('renders JSON for complex values', () => {
    const data = { config: { key: 'value', nested: true } }
    render(<DetailSlideOver data={data} onClose={mockOnClose} />)
    expect(screen.getByText('config')).toBeInTheDocument()
  })

  it('renders close button', () => {
    render(<DetailSlideOver data={{}} onClose={mockOnClose} />)
    expect(screen.getByText('Chiudi')).toBeInTheDocument()
  })

  it('calls onClose on close button click', () => {
    render(<DetailSlideOver data={{}} onClose={mockOnClose} />)
    fireEvent.click(screen.getByText('Chiudi'))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })
})
