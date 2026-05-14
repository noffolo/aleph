import { describe, it, expect, vi, beforeEach } from 'vitest'
import { reportError } from '../errorReporter'

const mockAddToast = vi.fn()

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn(),
    {
      getState: vi.fn(() => ({
        addToast: mockAddToast,
      })),
    },
  ),
}))

const originalEnv = { ...import.meta.env }

describe('reportError', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('console', { ...console, error: vi.fn() })
  })

  it('calls addToast with error type and context', () => {
    reportError('Network', 'connection failed')
    expect(mockAddToast).toHaveBeenCalledWith({
      type: 'error',
      context: 'Network',
      message: 'connection failed',
    })
  })

  it('extracts message from Error instance', () => {
    reportError('API', new Error('timeout'))
    expect(mockAddToast).toHaveBeenCalledWith({
      type: 'error',
      context: 'API',
      message: 'timeout',
    })
  })

  it('converts unknown error to string', () => {
    reportError('Parser', 42)
    expect(mockAddToast).toHaveBeenCalledWith({
      type: 'error',
      context: 'Parser',
      message: '42',
    })
  })

  it('handles null error gracefully', () => {
    reportError('Unknown', null)
    expect(mockAddToast).toHaveBeenCalledWith({
      type: 'error',
      context: 'Unknown',
      message: 'null',
    })
  })
})
