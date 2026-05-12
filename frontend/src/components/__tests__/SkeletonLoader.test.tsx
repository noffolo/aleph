import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { SkeletonLoader, SkeletonList } from '../SkeletonLoader'

describe('SkeletonLoader', () => {
  it('renders with default props (1 row, 1 col)', () => {
    const { container } = render(<SkeletonLoader />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper).toBeInTheDocument()
    expect(wrapper).toHaveClass('flex', 'flex-col', 'gap-3', 'w-full')

    const rows = wrapper.children
    expect(rows.length).toBe(1)
    const cols = rows[0].children
    expect(cols.length).toBe(1)
  })

  it('renders correct number of rows and columns', () => {
    const { container } = render(<SkeletonLoader rows={3} cols={2} />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.children.length).toBe(3)

    const firstRow = wrapper.children[0]
    expect(firstRow.children.length).toBe(2)
  })

  it('applies custom className', () => {
    const { container } = render(<SkeletonLoader className="my-loader" />)
    const wrapper = container.firstElementChild
    expect(wrapper).toHaveClass('my-loader')
  })

  it('renders skeleton bars with animate-pulse class', () => {
    const { container } = render(<SkeletonLoader rows={2} cols={1} />)
    const bars = container.querySelectorAll('.animate-pulse')
    expect(bars.length).toBe(2)
  })

  it('renders skeleton bars with correct base classes', () => {
    const { container } = render(<SkeletonLoader rows={1} cols={2} />)
    const bars = container.querySelectorAll('.animate-pulse')
    bars.forEach((bar) => {
      expect(bar).toHaveClass('h-4', 'bg-white/5', 'rounded-sm', 'flex-1')
    })
  })
})

describe('SkeletonList', () => {
  it('renders with default itemCount (5)', () => {
    const { container } = render(<SkeletonList />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper).toBeInTheDocument()
    expect(wrapper).toHaveClass('flex', 'flex-col', 'gap-2', 'w-full')
    expect(wrapper.children.length).toBe(5)
  })

  it('renders correct number of items', () => {
    const { container } = render(<SkeletonList itemCount={3} />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.children.length).toBe(3)
  })

  it('applies custom className', () => {
    const { container } = render(<SkeletonList className="my-list" />)
    const wrapper = container.firstElementChild
    expect(wrapper).toHaveClass('my-list')
  })

  it('renders list items with animate-pulse and rounded-md', () => {
    const { container } = render(<SkeletonList itemCount={2} />)
    const items = container.querySelectorAll('.animate-pulse')
    expect(items.length).toBe(2)
    items.forEach((item) => {
      expect(item).toHaveClass('rounded-md', 'bg-white/5')
    })
  })

  it('renders zero items when itemCount is 0', () => {
    const { container } = render(<SkeletonList itemCount={0} />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.children.length).toBe(0)
  })
})
