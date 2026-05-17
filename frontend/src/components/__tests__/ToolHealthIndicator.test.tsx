import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { ToolHealthIndicator } from '../ToolHealthIndicator';

describe('ToolHealthIndicator', () => {
  it('renders with healthy status', () => {
    const { container } = render(<ToolHealthIndicator status="healthy" />);
    const dot = container.querySelector('.bg-success');
    expect(dot).toBeInTheDocument();
    expect(dot).toHaveClass('animate-pulse');
  });

  it('renders with warning status', () => {
    const { container } = render(<ToolHealthIndicator status="warning" />);
    expect(container.querySelector('.bg-warning')).toBeInTheDocument();
  });

  it('renders with error status', () => {
    const { container } = render(<ToolHealthIndicator status="error" />);
    expect(container.querySelector('.bg-danger')).toBeInTheDocument();
  });

  it('renders with unknown status', () => {
    const { container } = render(<ToolHealthIndicator status="unknown" />);
    expect(container.querySelector('.bg-textDim')).toBeInTheDocument();
  });

  it('shows healthy tooltip text on hover', () => {
    const { container } = render(<ToolHealthIndicator status="healthy" lastCheck="2025-01-01" />);
    const tooltip = container.querySelector('.group-hover\\:block');
    expect(tooltip).toBeInTheDocument();
    expect(tooltip).toHaveTextContent('Sistemi Operativi');
    expect(tooltip).toHaveTextContent('2025-01-01');
  });

  it('shows warning tooltip text', () => {
    const { container } = render(<ToolHealthIndicator status="warning" />);
    const tooltip = container.querySelector('.group-hover\\:block');
    expect(tooltip).toHaveTextContent('Latenza Rilevata');
  });

  it('shows error tooltip text', () => {
    const { container } = render(<ToolHealthIndicator status="error" />);
    const tooltip = container.querySelector('.group-hover\\:block');
    expect(tooltip).toHaveTextContent('Errore di Esecuzione');
  });

  it('shows unknown tooltip text', () => {
    const { container } = render(<ToolHealthIndicator status="unknown" />);
    const tooltip = container.querySelector('.group-hover\\:block');
    expect(tooltip).toHaveTextContent('Stato Non Verificato');
  });

  it('renders with default unknown title when no lastCheck', () => {
    const { container } = render(<ToolHealthIndicator status="unknown" />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper).toHaveAttribute('title', 'Unknown status');
  });

  it('renders with lastCheck in title', () => {
    const { container } = render(<ToolHealthIndicator status="healthy" lastCheck="2025-06-15" />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper).toHaveAttribute('title', 'Last checked: 2025-06-15');
  });
});
