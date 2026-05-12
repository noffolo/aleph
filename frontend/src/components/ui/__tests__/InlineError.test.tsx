import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { InlineError } from '../InlineError'

describe('InlineError', () => {
  it('renders the error message', () => {
    render(<InlineError message="Something went wrong" />)
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('renders a container with border-danger classes', () => {
    const { container } = render(<InlineError message="error" />)
    const wrapper = container.firstElementChild
    expect(wrapper).toHaveClass('border-l-4', 'border-danger')
  })

  it('does not render dismiss button when onDismiss is not provided', () => {
    render(<InlineError message="error" />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('renders dismiss button when onDismiss is provided', () => {
    const onDismiss = vi.fn()
    render(<InlineError message="error" onDismiss={onDismiss} />)
    const button = screen.getByRole('button')
    expect(button).toBeInTheDocument()
  })

  it('calls onDismiss when dismiss button is clicked', () => {
    const onDismiss = vi.fn()
    render(<InlineError message="error" onDismiss={onDismiss} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('accepts and applies className via message container', () => {
    render(<InlineError message="test message" />)
    const msg = screen.getByText('test message')
    expect(msg).toHaveClass('text-body')
  })
})
