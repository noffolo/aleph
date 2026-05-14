import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { SystemScene } from '../SystemScene'

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['health', vi.fn()]),
}))

const mockSetSlideOverContent = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        return selector({
          slideOverContent: { type: 'health', title: 'Data Health' },
          setSlideOverContent: mockSetSlideOverContent,
        })
      }
      return { slideOverContent: { type: 'health', title: 'Data Health' }, setSlideOverContent: mockSetSlideOverContent }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({
        slideOverContent: { type: 'health', title: 'Data Health' },
        setSlideOverContent: mockSetSlideOverContent,
      })),
    },
  ),
}))

vi.mock('../../store/sceneMapping', () => ({
  SYSTEM_VIEWS: ['health', 'settings', 'predict'],
  VIEW_LABELS: { health: 'Data Health', settings: 'Settings', predict: 'Oracle' },
}))

vi.mock('../../components/SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
}))

describe('SystemScene', () => {
  it('renders the SkeletonLoader', () => {
    render(<SystemScene />)
    expect(screen.getByTestId('skeleton-loader')).toBeDefined()
  })

  it('renders within a flex container', () => {
    const { container } = render(<SystemScene />)
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper.className).toContain('flex')
  })

  it('sets slideOverContent via useEffect on mount', () => {
    render(<SystemScene />)
    expect(mockSetSlideOverContent).toHaveBeenCalled()
  })

  it('sets slideOverContent with correct type and title for default view (health)', () => {
    render(<SystemScene />)
    const call = mockSetSlideOverContent.mock.calls[0]?.[0]
    expect(call).toBeDefined()
    expect(call.type).toBe('health')
    expect(call.title).toBeDefined()
  })
})
