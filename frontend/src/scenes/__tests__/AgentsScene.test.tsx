import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { AgentsScene } from '../AgentsScene'

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['agent', vi.fn()]),
}))

const mockSetSlideOverContent = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        return selector({
          slideOverContent: { type: 'agent', title: 'Agents' },
          setSlideOverContent: mockSetSlideOverContent,
        })
      }
      return { slideOverContent: { type: 'agent', title: 'Agents' }, setSlideOverContent: mockSetSlideOverContent }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({
        slideOverContent: { type: 'agent', title: 'Agents' },
        setSlideOverContent: mockSetSlideOverContent,
      })),
    },
  ),
}))

vi.mock('../../store/sceneMapping', () => ({
  AGENT_VIEWS: ['agent', 'skill', 'tool', 'component'],
  VIEW_LABELS: { agent: 'Agents', skill: 'Skills', tool: 'Tools', component: 'Components' },
}))

vi.mock('../../components/SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
}))

describe('AgentsScene', () => {
  it('renders the SkeletonLoader', () => {
    render(<AgentsScene />)
    expect(screen.getByTestId('skeleton-loader')).toBeDefined()
  })

  it('renders within a flex container', () => {
    const { container } = render(<AgentsScene />)
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper.className).toContain('flex')
  })

  it('sets slideOverContent via useEffect on mount', () => {
    render(<AgentsScene />)
    expect(mockSetSlideOverContent).toHaveBeenCalled()
  })

  it('sets slideOverContent with correct type for default view (agent)', () => {
    render(<AgentsScene />)
    const call = mockSetSlideOverContent.mock.calls[0]?.[0]
    expect(call).toBeDefined()
    expect(call.type).toBe('agent')
  })
})
