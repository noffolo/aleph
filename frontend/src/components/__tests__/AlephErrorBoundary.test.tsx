import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { AlephErrorBoundary } from '../AlephErrorBoundary';

vi.mock('lucide-react', () => ({
  Binary: () => null,
}));

const { storeGetState } = vi.hoisted(() => ({
  storeGetState: vi.fn(() => ({
    setData: vi.fn(),
    setPredictions: vi.fn(),
    setLastError: vi.fn(),
    setCurrentView: vi.fn(),
  })),
}));

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(() => ({})), { getState: storeGetState }),
}));

describe('AlephErrorBoundary', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    storeGetState.mockReturnValue({
      setData: vi.fn(),
      setPredictions: vi.fn(),
      setLastError: vi.fn(),
      setCurrentView: vi.fn(),
    });
  });

  it('renders children when no error', () => {
    render(
      <AlephErrorBoundary>
        <div>child content</div>
      </AlephErrorBoundary>,
    );
    expect(screen.getByText('child content')).toBeInTheDocument();
  });

  it('catches rendering errors and shows fallback UI', () => {
    const ThrowComponent = () => {
      throw new Error('simulated render error');
    };

    render(
      <AlephErrorBoundary>
        <ThrowComponent />
      </AlephErrorBoundary>,
    );

    expect(screen.getByText(/Modalità Raw/i)).toBeInTheDocument();
    expect(screen.getByText(/Riprova/i)).toBeInTheDocument();
  });

  it('handleRetry calls store reset methods when clicked', () => {
    let shouldThrow = true;
    const ToggleComponent = () => {
      if (shouldThrow) throw new Error('click retry');
      return <div>after retry</div>;
    };

    render(
      <AlephErrorBoundary>
        <ToggleComponent />
      </AlephErrorBoundary>,
    );

    expect(screen.getByText(/Riprova/i)).toBeInTheDocument();

    const retryButton = screen.getByText(/Riprova/i);
    fireEvent.click(retryButton);

    expect(storeGetState).toHaveBeenCalled();
  });

  it('handleRetry does not throw when store mutations fail (empty catch regression)', () => {
    storeGetState.mockReturnValue({
      setData: vi.fn().mockImplementation(() => {
        throw new Error('store broken');
      }),
      setPredictions: vi.fn(),
      setLastError: vi.fn(),
      setCurrentView: vi.fn(),
    });

    let shouldThrow = true;
    const ToggleComponent = () => {
      if (shouldThrow) throw new Error('broken store test');
      return <div>recovered</div>;
    };

    render(
      <AlephErrorBoundary>
        <ToggleComponent />
      </AlephErrorBoundary>,
    );

    const retryButton = screen.getByText(/Riprova/i);

    expect(() => {
      fireEvent.click(retryButton);
    }).not.toThrow();
  });
});
