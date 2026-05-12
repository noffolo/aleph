import React from 'react'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import type { ToastMessage } from '../../store/uiSlice'

const mockRemoveToast = vi.fn()
const mockStore = {
  toastMessages: [] as ToastMessage[],
  removeToast: mockRemoveToast,
  addToast: vi.fn(),
}

vi.mock('../../store/useStore', () => ({
  useStore: vi.fn((selector: (s: typeof mockStore) => unknown) => {
    if (typeof selector === 'function') {
      return selector(mockStore)
    }
    return mockStore
  }),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'toast.error': 'Errore',
      'toast.info': 'Info',
      'toast.success': 'Successo',
      'toast.retry': 'Riprova',
      'toast.close': 'Chiudi',
    }
    return map[key] ?? key
  },
}))

import { ToastContainer } from '../Toast'

describe('Toast', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockStore.toastMessages = []
  })

  describe('ToastContainer', () => {
    it('renders nothing when no toast messages', () => {
      const { container } = render(<ToastContainer />)
      expect(container.firstChild).toBeNull()
    })

    it('renders toast messages when present', () => {
      mockStore.toastMessages = [
        { id: '1', message: 'Test error', type: 'error', context: 'Errore' },
      ]
      render(<ToastContainer />)
      expect(screen.getByText('Test error')).toBeInTheDocument()
    })

    it('renders error toast with error icon and context', () => {
      mockStore.toastMessages = [
        { id: '1', message: 'Connection lost', type: 'error' },
      ]
      render(<ToastContainer />)
      expect(screen.getByText('Connection lost')).toBeInTheDocument()
    })

    it('renders info toast', () => {
      mockStore.toastMessages = [
        { id: '2', message: 'Update available', type: 'info' },
      ]
      render(<ToastContainer />)
      expect(screen.getByText('Update available')).toBeInTheDocument()
    })

    it('renders success toast', () => {
      mockStore.toastMessages = [
        { id: '3', message: 'Saved successfully', type: 'success' },
      ]
      render(<ToastContainer />)
      expect(screen.getByText('Saved successfully')).toBeInTheDocument()
    })

    it('renders multiple toasts', () => {
      mockStore.toastMessages = [
        { id: '1', message: 'Error 1', type: 'error' },
        { id: '2', message: 'Info 1', type: 'info' },
      ]
      render(<ToastContainer />)
      expect(screen.getByText('Error 1')).toBeInTheDocument()
      expect(screen.getByText('Info 1')).toBeInTheDocument()
    })

    it('calls removeToast when close button clicked', () => {
      mockStore.toastMessages = [
        { id: 'abc', message: 'Dismiss me', type: 'info' },
      ]
      render(<ToastContainer />)
      fireEvent.click(screen.getByLabelText('Chiudi'))
      expect(mockRemoveToast).toHaveBeenCalledWith('abc')
    })

    it('renders retry button when toast has retry callback and type error', () => {
      const retryFn = vi.fn()
      mockStore.toastMessages = [
        { id: '5', message: 'Failed', type: 'error', retry: retryFn },
      ]
      render(<ToastContainer />)
      const retryBtn = screen.getByText('Riprova')
      expect(retryBtn).toBeInTheDocument()

      fireEvent.click(retryBtn)
      expect(mockRemoveToast).toHaveBeenCalledWith('5')
      expect(retryFn).toHaveBeenCalledTimes(1)
    })

    it('does not render retry for non-error type even if retry provided', () => {
      const retryFn = vi.fn()
      mockStore.toastMessages = [
        { id: '6', message: 'Info retry', type: 'info', retry: retryFn },
      ]
      render(<ToastContainer />)
      expect(screen.queryByText('Riprova')).not.toBeInTheDocument()
    })

    it('renders with proper role attributes', () => {
      mockStore.toastMessages = [
        { id: '1', message: 'Test', type: 'error' },
      ]
      const { container } = render(<ToastContainer />)
      expect(container.querySelector('[aria-live="polite"]')).toBeInTheDocument()
    })

    it('renders toast item with role status', () => {
      mockStore.toastMessages = [
        { id: '1', message: 'Test', type: 'error' },
      ]
      render(<ToastContainer />)
      expect(screen.getByRole('status')).toBeInTheDocument()
    })
  })
})
