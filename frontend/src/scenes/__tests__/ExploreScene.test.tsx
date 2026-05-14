import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ExploreScene } from '../ExploreScene'

vi.mock('nuqs', () => ({
  useQueryState: vi.fn(() => ['explore', vi.fn()]),
}))

const mockSetSlideOverContent = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        return selector({
          slideOverContent: { type: 'explore', title: 'Explorer' },
          setSlideOverContent: mockSetSlideOverContent,
        })
      }
      return { slideOverContent: { type: 'explore', title: 'Explorer' }, setSlideOverContent: mockSetSlideOverContent }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({
        slideOverContent: { type: 'explore', title: 'Explorer' },
        setSlideOverContent: mockSetSlideOverContent,
      })),
    },
  ),
}))

vi.mock('../../store/sceneMapping', () => ({
  EXPLORE_VIEWS: ['explore', 'library', 'ontology', 'data'],
  VIEW_LABELS: { explore: 'Explorer', library: 'Library', ontology: 'Ontologies', data: 'Data Sources' },
}))

vi.mock('../../components/SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
}))

describe('ExploreScene', () => {
  it('renders the SkeletonLoader', () => {
    render(<ExploreScene />)
    expect(screen.getByTestId('skeleton-loader')).toBeDefined()
  })

  it('sets slideOverContent via useEffect on mount', () => {
    render(<ExploreScene />)
    expect(mockSetSlideOverContent).toHaveBeenCalled()
  })

  it('sets slideOverContent with correct type for default view (explore)', () => {
    render(<ExploreScene />)
    const call = mockSetSlideOverContent.mock.calls[0]?.[0]
    expect(call).toBeDefined()
    expect(call.type).toBe('explore')
  })
})
