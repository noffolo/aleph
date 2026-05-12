import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Switch } from '../switch'

describe('Switch', () => {
  it('renders a switch element', () => {
    const { container } = render(<Switch />)
    const switchEl = container.querySelector('[data-slot="switch"]')
    expect(switchEl).toBeInTheDocument()
  })

  it('has default size classes', () => {
    const { container } = render(<Switch />)
    const switchEl = container.querySelector('[data-slot="switch"]')
    expect(switchEl).toHaveAttribute('data-size', 'default')
  })

  it('has sm size when specified', () => {
    const { container } = render(<Switch size="sm" />)
    const switchEl = container.querySelector('[data-slot="switch"]')
    expect(switchEl).toHaveAttribute('data-size', 'sm')
  })

  it('applies custom className', () => {
    const { container } = render(<Switch className="my-switch" />)
    const switchEl = container.querySelector('[data-slot="switch"]')
    expect(switchEl).toHaveClass('my-switch')
  })

  it('renders as disabled', () => {
    const { container } = render(<Switch disabled />)
    const switchEl = container.querySelector('[data-slot="switch"]')
    expect(switchEl).toHaveClass('data-disabled:cursor-not-allowed')
  })

  it('renders a thumb element', () => {
    const { container } = render(<Switch />)
    const thumb = container.querySelector('[data-slot="switch-thumb"]')
    expect(thumb).toBeInTheDocument()
  })

  it('renders thumb with rounded-full and bg-background classes', () => {
    const { container } = render(<Switch />)
    const thumb = container.querySelector('[data-slot="switch-thumb"]')
    expect(thumb).toHaveClass('rounded-full', 'bg-background')
  })
})
