import React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Input } from '../input'

describe('Input', () => {
  it('renders an input element', () => {
    const { container } = render(<Input />)
    const input = container.querySelector('input')
    expect(input).toBeInTheDocument()
  })

  it('applies default classes', () => {
    const { container } = render(<Input />)
    const input = container.querySelector('input')
    expect(input).toHaveClass('h-8', 'w-full', 'min-w-0', 'rounded-lg')
  })

  it('applies custom className', () => {
    const { container } = render(<Input className="custom-input" />)
    const input = container.querySelector('input')
    expect(input).toHaveClass('custom-input')
  })

  it('renders as disabled', () => {
    const { container } = render(<Input disabled />)
    const input = container.querySelector('input')
    expect(input?.disabled).toBe(true)
    expect(input).toHaveClass('disabled:pointer-events-none')
  })

  it('sets placeholder', () => {
    const { container } = render(<Input placeholder="Enter text" />)
    const input = container.querySelector('input')
    expect(input).toHaveAttribute('placeholder', 'Enter text')
  })

  it('sets type attribute', () => {
    const { container } = render(<Input type="password" />)
    const input = container.querySelector('input')
    expect(input).toHaveAttribute('type', 'password')
  })

  it('has data-slot="input" attribute', () => {
    const { container } = render(<Input />)
    const input = container.querySelector('input')
    expect(input).toHaveAttribute('data-slot', 'input')
  })

  it('accepts additional props', () => {
    const { container } = render(<Input id="test-id" name="test-name" />)
    const input = container.querySelector('input')
    expect(input).toHaveAttribute('id', 'test-id')
    expect(input).toHaveAttribute('name', 'test-name')
  })
})
