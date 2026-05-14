import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Sidebar } from '../Sidebar'
import { isEnabled } from '../../config/features'

const mockSetCurrentScene = vi.fn()
const mockSetCurrentView = vi.fn()
const mockSetSlideOverContent = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((selector?: (state: unknown) => unknown) => {
      if (typeof selector === 'function') {
        return selector({
          slideOverContent: null,
          currentScene: 'explore',
          currentView: 'explore',
          setCurrentScene: mockSetCurrentScene,
          setCurrentView: mockSetCurrentView,
          setSlideOverContent: mockSetSlideOverContent,
        })
      }
      return {
        slideOverContent: null,
        currentScene: 'explore',
        currentView: 'explore',
        setCurrentScene: mockSetCurrentScene,
        setCurrentView: mockSetCurrentView,
        setSlideOverContent: mockSetSlideOverContent,
      }
    }),
    {
      subscribe: vi.fn(() => vi.fn()),
      getState: vi.fn(() => ({
        setCurrentScene: mockSetCurrentScene,
        setCurrentView: mockSetCurrentView,
        setSlideOverContent: mockSetSlideOverContent,
      })),
    },
  ),
}))

vi.mock('../../config/features', () => ({
  isEnabled: vi.fn(),
  FEATURE_COMPACT_SIDEBAR: 'FEATURE_COMPACT_SIDEBAR',
}))

const mockedIsEnabled = vi.mocked(isEnabled)

vi.mock('lucide-react', () => {
  const createIcon = (name: string) => () => React.createElement('div', { 'data-testid': `icon-${name}` })
  return {
    LayoutGrid: createIcon('LayoutGrid'),
    Binary: createIcon('Binary'),
    Activity: createIcon('Activity'),
    Bot: createIcon('Bot'),
    Eye: createIcon('Eye'),
    Book: createIcon('Book'),
    Compass: createIcon('Compass'),
    Cpu: createIcon('Cpu'),
    Database: createIcon('Database'),
    Gauge: createIcon('Gauge'),
    Package: createIcon('Package'),
    Sliders: createIcon('Sliders'),
    Monitor: createIcon('Monitor'),
    Settings: createIcon('Settings'),
    Terminal: createIcon('Terminal'),
    Wrench: createIcon('Wrench'),
    Zap: createIcon('Zap'),
    Users: createIcon('Users'),
  }
})

vi.mock('../../i18n', () => ({
  t: (key: string) => key,
}))

describe('Sidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedIsEnabled.mockReturnValue(true)
  })

  const defaultProps = { projectID: 'proj-123', onShowOnboarding: vi.fn() }

  it('renders the navigation', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByLabelText('Main navigation')).toBeDefined()
  })

  it('renders the Aleph logo (Binary icon)', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByTestId('icon-Binary')).toBeDefined()
  })

  it('renders Copilot button', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByLabelText('Copilot')).toBeDefined()
  })

  it('triggers onShowOnboarding when clicking without projectID', () => {
    const onShowOnboarding = vi.fn()
    render(<Sidebar projectID="" onShowOnboarding={onShowOnboarding} />)
    fireEvent.click(screen.getByLabelText('Explore'))
    expect(onShowOnboarding).toHaveBeenCalled()
  })

  it('navigates to explore scene on Explore click', () => {
    render(<Sidebar {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Explore'))
    expect(mockSetCurrentScene).toHaveBeenCalledWith('explore')
  })

  it('navigates to agents scene on Agents click', () => {
    render(<Sidebar {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Agents'))
    expect(mockSetCurrentScene).toHaveBeenCalledWith('agents')
  })

  it('navigates to system scene on System click', () => {
    render(<Sidebar {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('System'))
    expect(mockSetCurrentScene).toHaveBeenCalledWith('system')
  })

  it('navigates to terminal scene on Dashboard click (ALL mode)', () => {
    mockedIsEnabled.mockReturnValue(false)
    render(<Sidebar {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Dashboard'))
    expect(mockSetCurrentScene).toHaveBeenCalledWith('terminal')
  })

  it('opens copilot when clicking Copilot button', () => {
    render(<Sidebar {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Copilot'))
    expect(mockSetCurrentView).toHaveBeenCalledWith('copilot')
    expect(mockSetSlideOverContent).toHaveBeenCalledWith(null)
  })

  it('sets slideOverContent to the correct type when clicking a mapped item', () => {
    render(<Sidebar {...defaultProps} />)
    fireEvent.click(screen.getByLabelText('Agents'))
    expect(mockSetSlideOverContent).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'agent' })
    )
  })

  it('renders Settings button', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByLabelText('proj-123')).toBeDefined()
  })

  it('renders in compact mode when FEATURE_COMPACT_SIDEBAR is enabled', () => {
    mockedIsEnabled.mockReturnValue(true)
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByLabelText('Terminal')).toBeDefined()
    expect(screen.getByLabelText('Explore')).toBeDefined()
    expect(screen.getByLabelText('Agents')).toBeDefined()
    expect(screen.getByLabelText('System')).toBeDefined()
  })

  it('renders all items when FEATURE_COMPACT_SIDEBAR is disabled', () => {
    mockedIsEnabled.mockReturnValue(false)
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByLabelText('Dashboard')).toBeDefined()
    expect(screen.getByLabelText('Data Health')).toBeDefined()
    expect(screen.getByLabelText('Library')).toBeDefined()
    expect(screen.getByLabelText('Skills')).toBeDefined()
    expect(screen.getByLabelText('Tools')).toBeDefined()
    expect(screen.getByLabelText('Components')).toBeDefined()
    expect(screen.getByLabelText('Settings')).toBeDefined()
  })
})
