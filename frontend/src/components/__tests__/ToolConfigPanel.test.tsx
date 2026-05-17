import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToolConfigPanel } from '../ToolConfigPanel';

vi.mock('../../lib/errorReporter', () => ({
  reportError: vi.fn(),
}));

vi.mock('lucide-react', () => ({
  Copy: () => <span data-testid="icon-copy">Copy</span>,
  Check: () => <span data-testid="icon-check">Check</span>,
}));

const writeTextMock = vi.fn();

beforeAll(() => {
  if (!navigator.clipboard) {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: writeTextMock },
      writable: true,
      configurable: true,
    });
  }
});

describe('ToolConfigPanel', () => {
  const config = { name: 'TestTool', version: '1.0.0', enabled: true };

  beforeEach(() => {
    vi.clearAllMocks();
    writeTextMock.mockResolvedValue(undefined);
  });

  it('renders the title', () => {
    render(<ToolConfigPanel config={config} title="My Config" />);
    expect(screen.getByText('My Config')).toBeInTheDocument();
  });

  it('renders default title when not provided', () => {
    render(<ToolConfigPanel config={config} />);
    expect(screen.getByText('Configurazione Strumento')).toBeInTheDocument();
  });

  it('renders config as formatted JSON', () => {
    render(<ToolConfigPanel config={config} />);
    expect(screen.getByText(/TestTool/)).toBeInTheDocument();
    expect(screen.getByText(/"name"/)).toBeInTheDocument();
  });

  it('shows copy button', () => {
    render(<ToolConfigPanel config={config} />);
    expect(screen.getByTitle('Copia JSON')).toBeInTheDocument();
  });

  it('copies and shows check icon on button click', async () => {
    const user = userEvent.setup();
    writeTextMock.mockResolvedValue(undefined);

    render(<ToolConfigPanel config={config} />);
    await user.click(screen.getByTitle('Copia JSON'));
    expect(screen.getByTestId('icon-check')).toBeInTheDocument();
  });

  it('renders empty config object', () => {
    render(<ToolConfigPanel config={{}} />);
    expect(screen.getByText('{}')).toBeInTheDocument();
  });

  it('renders config with nested objects', () => {
    const nested = { a: { b: { c: 'deep' } } };
    render(<ToolConfigPanel config={nested} />);
    expect(screen.getByText(/"deep"/)).toBeInTheDocument();
  });
});
