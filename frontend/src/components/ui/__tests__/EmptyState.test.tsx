import React from 'react'
import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { EmptyState } from '../EmptyState'

describe('EmptyState', () => {
  it('renders a container div with w-full and h-full classes', () => {
    const { container } = render(<EmptyState />)
    const wrapper = container.firstElementChild
    expect(wrapper).toBeInTheDocument()
    expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center', 'py-8', 'w-full', 'h-full')
  })

  it('renders a message element with correct classes', () => {
    const { container } = render(<EmptyState />)
    const msgEl = container.querySelector('.text-textMuted.font-mono.text-center.text-meta')
    expect(msgEl).toBeInTheDocument()
  })

  it('renders a message that is visible', () => {
    const { container } = render(<EmptyState />)
    const msgEl = container.querySelector('.text-textMuted.font-mono.text-center.text-meta')
    expect(msgEl).toHaveClass('opacity-100')
  })

  it('renders children structure correctly', () => {
    const { container } = render(<EmptyState />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.children.length).toBe(1)
    const inner = wrapper.children[0]
    expect(inner).toHaveClass('text-textMuted', 'font-mono', 'text-center', 'text-meta')
  })
})
