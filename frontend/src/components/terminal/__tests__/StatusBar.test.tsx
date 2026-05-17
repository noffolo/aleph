import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatusBar } from '../StatusBar';

const { mockGetState } = vi.hoisted(() => ({
  mockGetState: vi.fn(() => ({
    slideOverContent: null,
    inputMode: false,
  })),
}));

vi.mock('../../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((sel: (s: Record<string, unknown>) => unknown) => sel(mockGetState())),
    { getState: mockGetState }
  ),
}));

describe('StatusBar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders ALEPH brand', () => {
    render(<StatusBar projectID="proj-1" ollamaHealthy={true} nlpHealthy={true} />);
    expect(screen.getByText('ALEPH')).toBeInTheDocument();
  });

  it('renders project ID', () => {
    render(<StatusBar projectID="my-project" ollamaHealthy={true} nlpHealthy={true} />);
    expect(screen.getByText('my-project')).toBeInTheDocument();
  });

  it('renders NO PROJECT when projectID is empty', () => {
    render(<StatusBar projectID="" ollamaHealthy={true} nlpHealthy={true} />);
    expect(screen.getByText('NO PROJECT')).toBeInTheDocument();
  });

  it('shows READY context when no slide over', () => {
    render(<StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={true} />);
    expect(screen.getByText('READY')).toBeInTheDocument();
  });

  it('shows [CMD] when inputMode is false', () => {
    render(<StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={true} />);
    expect(screen.getByText('[CMD]')).toBeInTheDocument();
  });

  it('shows healthy OLLAMA indicator', () => {
    const { container } = render(
      <StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={true} />
    );
    expect(screen.getByText('OLLAMA')).toBeInTheDocument();
    const ollamaDot = container.querySelector('.bg-success');
    expect(ollamaDot).toBeInTheDocument();
  });

  it('shows unhealthy OLLAMA indicator', () => {
    const { container } = render(
      <StatusBar projectID="p" ollamaHealthy={false} nlpHealthy={true} />
    );
    expect(screen.getByText('OLLAMA')).toBeInTheDocument();
    const ollamaDot = container.querySelector('.bg-danger');
    expect(ollamaDot).toBeInTheDocument();
  });

  it('shows healthy NLP indicator', () => {
    const { container } = render(
      <StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={true} />
    );
    expect(screen.getByText('NLP')).toBeInTheDocument();
    const nlpDot = container.querySelector('.bg-primary');
    expect(nlpDot).toBeInTheDocument();
  });

  it('shows unhealthy NLP indicator', () => {
    const { container } = render(
      <StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={false} />
    );
    expect(screen.getByText('NLP')).toBeInTheDocument();
    const nlpDot = container.querySelector('.bg-warning');
    expect(nlpDot).toBeInTheDocument();
  });

  it('has status role for accessibility', () => {
    render(<StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={true} />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('has aria-live polite', () => {
    render(<StatusBar projectID="p" ollamaHealthy={true} nlpHealthy={true} />);
    const status = screen.getByRole('status');
    expect(status).toHaveAttribute('aria-live', 'polite');
  });
});
