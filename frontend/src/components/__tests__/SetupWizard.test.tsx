import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SetupWizard } from '../SetupWizard'

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'setup.copied': 'Copied!',
      'setup.copy': 'Copy',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => ({
  Database: () => null,
  Zap: () => null,
  ArrowRight: () => null,
  CheckCircle2: () => null,
  ShieldCheck: () => null,
  Key: () => null,
  Copy: () => null,
  Activity: () => null,
  AlertTriangle: () => null,
}))

describe('SetupWizard', () => {
  const mockOnComplete = vi.fn()
  const mockOnLogin = vi.fn()
  const mockOnCreateProject = vi.fn()
  const mockOnCreateApiKey = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders step 0 (login) by default', () => {
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    expect(screen.getByText('Connettiti ad Aleph')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Inserisci la tua API Key')).toBeInTheDocument()
  })

  it('advances to step 1 after successful login', async () => {
    mockOnLogin.mockResolvedValueOnce(undefined)
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    const input = screen.getByPlaceholderText('Inserisci la tua API Key')
    fireEvent.change(input, { target: { value: 'test-key' } })
    fireEvent.click(screen.getByText('Connettiti'))
    await waitFor(() => {
      expect(screen.getByText('Crea il tuo spazio di lavoro')).toBeInTheDocument()
    })
  })

  it('shows error on failed login', async () => {
    mockOnLogin.mockRejectedValueOnce(new Error('Invalid API key'))
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    const input = screen.getByPlaceholderText('Inserisci la tua API Key')
    fireEvent.change(input, { target: { value: 'bad-key' } })
    fireEvent.click(screen.getByText('Connettiti'))
    await waitFor(() => {
      expect(screen.getByText('Invalid API key')).toBeInTheDocument()
    })
  })

  it('advances through all steps to completion', async () => {
    mockOnLogin.mockResolvedValueOnce(undefined)
    mockOnCreateProject.mockResolvedValueOnce('proj-123')
    mockOnCreateApiKey.mockResolvedValueOnce('secret-key-abc')

    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )

    fireEvent.change(screen.getByPlaceholderText('Inserisci la tua API Key'), {
      target: { value: 'test-key' },
    })
    fireEvent.click(screen.getByText('Connettiti'))
    await waitFor(() => {
      expect(screen.getByText('Crea il tuo spazio di lavoro')).toBeInTheDocument()
    })

    const projectInput = screen.getByPlaceholderText('workspace-name')
    fireEvent.change(projectInput, { target: { value: 'my-project' } })
    fireEvent.click(screen.getByText('Prosegui'))
    await waitFor(() => {
      expect(screen.getByText('Proteggi lo spazio di lavoro')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('Genera API Key Protetta'))
    await waitFor(() => {
      expect(screen.getByText('Spazio di lavoro pronto')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('Inizia'))
    expect(mockOnComplete).toHaveBeenCalledWith('proj-123', 'secret-key-abc')
  })

  it('renders language toggle (IT/EN)', () => {
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    expect(screen.getByText('IT')).toBeInTheDocument()
    expect(screen.getByText('EN')).toBeInTheDocument()
  })

  it('renders step indicators correctly', () => {
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('4')).toBeInTheDocument()
  })

  it('switches to English language', () => {
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    fireEvent.click(screen.getByText('EN'))
    expect(screen.getByText('Connect to Aleph')).toBeInTheDocument()
  })

  it('submits login on Enter key', async () => {
    mockOnLogin.mockResolvedValueOnce(undefined)
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    const input = screen.getByPlaceholderText('Inserisci la tua API Key')
    fireEvent.change(input, { target: { value: 'enter-key' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => {
      expect(mockOnLogin).toHaveBeenCalledWith('enter-key')
    })
  })

  it('shows error when project creation fails', async () => {
    mockOnLogin.mockResolvedValueOnce(undefined)
    mockOnCreateProject.mockRejectedValueOnce(new Error('Name taken'))
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    fireEvent.change(screen.getByPlaceholderText('Inserisci la tua API Key'), {
      target: { value: 'test-key' },
    })
    fireEvent.click(screen.getByText('Connettiti'))
    await waitFor(() => {
      expect(screen.getByText('Crea il tuo spazio di lavoro')).toBeInTheDocument()
    })
    const projectInput = screen.getByPlaceholderText('workspace-name')
    fireEvent.change(projectInput, { target: { value: 'bad-project' } })
    fireEvent.click(screen.getByText('Prosegui'))
    await waitFor(() => {
      expect(screen.getByText('Name taken')).toBeInTheDocument()
    })
  })

  it('shows error when API key generation fails', async () => {
    mockOnLogin.mockResolvedValueOnce(undefined)
    mockOnCreateProject.mockResolvedValueOnce('proj-err')
    mockOnCreateApiKey.mockRejectedValueOnce(new Error('Key generation failed'))
    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    fireEvent.change(screen.getByPlaceholderText('Inserisci la tua API Key'), {
      target: { value: 'test-key' },
    })
    fireEvent.click(screen.getByText('Connettiti'))
    await waitFor(() => {
      expect(screen.getByText('Crea il tuo spazio di lavoro')).toBeInTheDocument()
    })
    fireEvent.change(screen.getByPlaceholderText('workspace-name'), {
      target: { value: 'err-project' },
    })
    fireEvent.click(screen.getByText('Prosegui'))
    await waitFor(() => {
      expect(screen.getByText('Genera API Key Protetta')).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText('Genera API Key Protetta'))
    await waitFor(() => {
      expect(screen.getByText('Key generation failed')).toBeInTheDocument()
    })
  })

  it('copies API key to clipboard in final step', async () => {
    const writeTextSpy = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText: writeTextSpy } })
    vi.spyOn(window, 'alert').mockImplementation(() => {})

    mockOnLogin.mockResolvedValueOnce(undefined)
    mockOnCreateProject.mockResolvedValueOnce('proj-c')
    mockOnCreateApiKey.mockResolvedValueOnce('secret-copy-key')

    render(
      <SetupWizard
        onComplete={mockOnComplete}
        onLogin={mockOnLogin}
        onCreateProject={mockOnCreateProject}
        onCreateApiKey={mockOnCreateApiKey}
      />,
    )
    fireEvent.change(screen.getByPlaceholderText('Inserisci la tua API Key'), {
      target: { value: 'test-key' },
    })
    fireEvent.click(screen.getByText('Connettiti'))
    await waitFor(() => {
      expect(screen.getByText('Crea il tuo spazio di lavoro')).toBeInTheDocument()
    })
    fireEvent.change(screen.getByPlaceholderText('workspace-name'), {
      target: { value: 'copy-project' },
    })
    fireEvent.click(screen.getByText('Prosegui'))
    await waitFor(() => {
      expect(screen.getByText('Genera API Key Protetta')).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText('Genera API Key Protetta'))
    await waitFor(() => {
      expect(screen.getByText('Spazio di lavoro pronto')).toBeInTheDocument()
    })
    const copyBtn = screen.getByTitle('Copy')
    fireEvent.click(copyBtn)
    await waitFor(() => {
      expect(writeTextSpy).toHaveBeenCalledWith('secret-copy-key')
    })
  })
})
