import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

const mockToggleSection = vi.fn()
const mockSetEnableScanline = vi.fn()
const mockSetEnableGlow = vi.fn()
const mockSetEnableFlicker = vi.fn()

const mockStore = () => ({
  enableScanline: true,
  enableGlow: false,
  enableFlicker: true,
  expandedSections: { 'settings.quick': true, 'settings.all': false, 'settings.advanced': false },
  toggleSection: mockToggleSection,
  setEnableScanline: mockSetEnableScanline,
  setEnableGlow: mockSetEnableGlow,
  setEnableFlicker: mockSetEnableFlicker,
})

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((sel: (s: ReturnType<typeof mockStore>) => unknown) => sel(mockStore())),
    {
      getState: () => ({
        setEnableScanline: mockSetEnableScanline,
        setEnableGlow: mockSetEnableGlow,
        setEnableFlicker: mockSetEnableFlicker,
      }),
    },
  ),
}))

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'settings.title': 'Impostazioni',
      'settings.scanlines': 'Linee di scansione',
      'settings.glow': 'Effetto glow',
      'settings.flicker': 'Effetto flicker',
      'settings.revoke': 'Revoca',
      'settings.sendTest': 'Invia test',
      'settings.webhookSecret': 'Webhook secret',
      'settings.apiKey': 'Chiavi API',
    }
    return map[key] ?? key
  },
}))

vi.mock('lucide-react', () => {
  const Icon = (name: string) => {
    const Comp = (props: React.SVGProps<SVGSVGElement>) => <svg {...props} data-testid={`icon-${name}`} />
    Comp.displayName = name
    return Comp
  }
  return {
    Key: Icon('Key'),
    Plus: Icon('Plus'),
    Trash2: Icon('Trash2'),
    Bell: Icon('Bell'),
    Globe: Icon('Globe'),
    Shield: Icon('Shield'),
    Monitor: Icon('Monitor'),
    Sparkles: Icon('Sparkles'),
    ScanLine: Icon('ScanLine'),
    Settings2: Icon('Settings2'),
  }
})

vi.mock('../SkeletonLoader', () => ({
  SkeletonLoader: () => <div data-testid="skeleton-loader">Loading...</div>,
}))

vi.mock('../ui/InlineError', () => ({
  InlineError: ({ message }: { message: string }) => <div data-testid="inline-error">{message}</div>,
}))

vi.mock('../ui/GlassPanel', () => ({
  GlassPanel: ({
    header,
    children,
    sectionKey,
    expanded,
    onToggle,
    advanced,
    showAdvanced,
  }: {
    header: string
    children: React.ReactNode
    sectionKey: string
    expanded: boolean
    onToggle: () => void
    advanced?: boolean
    showAdvanced?: boolean
  }) => {
    if (advanced && !showAdvanced) return null
    return (
      <div data-testid={`glass-panel-${sectionKey}`} data-expanded={expanded}>
        <button data-testid={`toggle-${sectionKey}`} onClick={onToggle}>{header}</button>
        {children}
      </div>
    )
  },
}))

import { SettingsView } from '../SettingsView'

const makeApiKey = (overrides?: Record<string, unknown>) => ({
  id: 'k1',
  label: 'My Key',
  key: 'sk-abcdefgh12345678',
  createdAt: 1700000000,
  ...overrides,
})

const defaultProps = {
  apiKeys: [] as ReturnType<typeof makeApiKey>[],
  notificationChannels: [] as { id: string; name: string; type: string; configJson: string }[],
  onCreateApiKey: vi.fn(),
  onDeleteApiKey: vi.fn(),
  onSendWebhook: vi.fn(),
}

describe('SettingsView', () => {
  let props: typeof defaultProps

  beforeEach(() => {
    vi.clearAllMocks()
    props = {
      ...defaultProps,
      onCreateApiKey: vi.fn(),
      onDeleteApiKey: vi.fn(),
      onSendWebhook: vi.fn(),
    }
  })

  // --- Render states ---
  it('renders skeleton when isLoading is true', () => {
    render(<SettingsView {...props} isLoading={true} />)
    expect(screen.getByTestId('skeleton-loader')).toBeInTheDocument()
  })

  it('renders error when error is provided', () => {
    render(<SettingsView {...props} error="Service down" />)
    expect(screen.getByTestId('inline-error')).toHaveTextContent('Service down')
  })

  it('renders settings region with aria-label', () => {
    render(<SettingsView {...props} />)
    expect(screen.getByRole('region', { name: 'Settings' })).toBeInTheDocument()
  })

  it('renders title', () => {
    render(<SettingsView {...props} />)
    expect(screen.getByText('Impostazioni')).toBeInTheDocument()
  })

  // --- Toggle switches ---
  it('renders scanline toggle and calls setEnableScanline on click', () => {
    render(<SettingsView {...props} />)
    const toggle = screen.getByLabelText('Toggle scanlines')
    expect(toggle).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(toggle)
    expect(mockSetEnableScanline).toHaveBeenCalledWith(false)
  })

  it('renders glow toggle and calls setEnableGlow on click', () => {
    render(<SettingsView {...props} />)
    const toggle = screen.getByLabelText('Toggle glow effect')
    expect(toggle).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(toggle)
    expect(mockSetEnableGlow).toHaveBeenCalledWith(true)
  })

  it('renders flicker toggle and calls setEnableFlicker on click', () => {
    render(<SettingsView {...props} />)
    const toggle = screen.getByLabelText('Toggle flicker effect')
    expect(toggle).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(toggle)
    expect(mockSetEnableFlicker).toHaveBeenCalledWith(false)
  })

  // --- API Keys ---
  it('renders API key section with key info', () => {
    const apiKeys = [makeApiKey()]
    render(<SettingsView {...props} apiKeys={apiKeys} />)
    expect(screen.getByText('My Key')).toBeInTheDocument()
    expect(screen.getByText('sk-a••••••')).toBeInTheDocument()
  })

  it('shows empty API keys message when none provided', () => {
    render(<SettingsView {...props} />)
    expect(screen.getByText('Nessuna chiave API configurata')).toBeInTheDocument()
  })

  it('calls onDeleteApiKey after confirmation', () => {
    window.confirm = vi.fn(() => true)
    const apiKeys = [makeApiKey()]
    render(<SettingsView {...props} apiKeys={apiKeys} />)
    fireEvent.click(screen.getByLabelText('Revoca'))
    expect(props.onDeleteApiKey).toHaveBeenCalledWith('k1')
  })

  // --- Notification channels ---
  it('renders notification channels', () => {
    const channels = [{ id: 'ch1', name: 'Slack', type: 'webhook', configJson: '{}' }]
    render(<SettingsView {...props} notificationChannels={channels} />)
    expect(screen.getByText('Slack')).toBeInTheDocument()
    expect(screen.getByText('webhook')).toBeInTheDocument()
  })

  it('shows empty channels message when none provided', () => {
    render(<SettingsView {...props} />)
    expect(screen.getByText('Nessun canale configurato')).toBeInTheDocument()
  })

  // --- Webhook test ---
  it('renders webhook test form inputs', () => {
    render(<SettingsView {...props} />)
    expect(screen.getByPlaceholderText('https://hooks.example.com/...')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('{"event": "test"}')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Webhook secret')).toBeInTheDocument()
  })

  it('disables send test button when URL is empty', () => {
    render(<SettingsView {...props} />)
    const btn = screen.getByLabelText('Send test webhook')
    expect(btn).toBeDisabled()
  })

  it('calls onSendWebhook with form data on click', () => {
    render(<SettingsView {...props} />)
    fireEvent.change(screen.getByPlaceholderText('https://hooks.example.com/...'), {
      target: { value: 'https://hooks.test.com/webhook' },
    })
    fireEvent.change(screen.getByPlaceholderText('{"event": "test"}'), {
      target: { value: '{"event": "deploy"}' },
    })
    fireEvent.change(screen.getByPlaceholderText('Webhook secret'), {
      target: { value: 'secret123' },
    })
    fireEvent.click(screen.getByLabelText('Send test webhook'))
    expect(props.onSendWebhook).toHaveBeenCalledWith(
      'https://hooks.test.com/webhook',
      '{"event": "deploy"}',
      'secret123',
    )
  })

  // --- Advanced settings toggle ---
  it('toggles advanced settings panel', () => {
    render(<SettingsView {...props} />)
    const toggleBtn = screen.getByLabelText('Toggle advanced settings')
    fireEvent.click(toggleBtn)
    // After click, showAdvanced should be true and advanced panel should appear
    expect(screen.getByTestId('glass-panel-settings.advanced')).toBeInTheDocument()
  })

  // --- Inline mode ---
  it('applies inline styling when inline prop is true', () => {
    const { container } = render(<SettingsView {...props} inline={true} />)
    const root = container.firstElementChild as HTMLElement
    expect(root.className).not.toContain('max-w-4xl')
  })

  // --- Section toggling ---
  it('toggles sections via GlassPanel onToggle', () => {
    render(<SettingsView {...props} />)
    fireEvent.click(screen.getByTestId('toggle-settings.quick'))
    expect(mockToggleSection).toHaveBeenCalledWith('settings.quick')
  })

  it('renders masked key with fallback when key is empty', () => {
    const apiKeys = [makeApiKey({ key: '' })]
    render(<SettingsView {...props} apiKeys={apiKeys} />)
    expect(screen.getByText('••••••••')).toBeInTheDocument()
  })

  it('renders api key info in quick settings when apiKeys not empty', () => {
    const apiKeys = [makeApiKey()]
    render(<SettingsView {...props} apiKeys={apiKeys} />)
    expect(screen.getByText('sk-a••••••')).toBeInTheDocument()
  })

  it('renders developer panel content when showAdvanced is true', () => {
    render(<SettingsView {...props} />)
    fireEvent.click(screen.getByLabelText('Toggle advanced settings'))
    expect(screen.getByText('Debug Logging')).toBeInTheDocument()
    expect(screen.getByText('Feature Flags')).toBeInTheDocument()
    expect(screen.getByText('DuckDB Query Inspector')).toBeInTheDocument()
  })
})
