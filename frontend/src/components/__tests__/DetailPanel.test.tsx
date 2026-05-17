import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DetailPanel } from '../DetailPanel';
import type { Row } from '../../store/types';

vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'detail.title': 'Dettaglio Record',
      'detail.empty': 'Vuoto',
      'detail.noData': 'Nessun dato',
    };
    return map[key] ?? key;
  },
}));

vi.mock('lucide-react', () => ({
  X: () => <span data-testid="icon-x">X</span>,
}));

describe('DetailPanel', () => {
  it('returns null when no selectedRow', () => {
    const { container } = render(
      <DetailPanel selectedRow={null} onClose={() => {}} />
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders when selectedRow is provided', () => {
    const row: Row = { values: { name: 'Alice' } };
    const { container } = render(
      <DetailPanel selectedRow={row} onClose={() => {}} />
    );
    expect(container.firstChild).toBeInTheDocument();
  });

  it('renders row values', () => {
    const row: Row = { values: { name: 'Alice', age: 30 } };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.getByText('name')).toBeInTheDocument();
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('age')).toBeInTheDocument();
    expect(screen.getByText('30')).toBeInTheDocument();
  });

  it('shows empty text for empty string values', () => {
    const row: Row = { values: { name: '' } };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.getByText('Vuoto')).toBeInTheDocument();
  });

  it('shows empty text for null values', () => {
    const row: Row = { values: { name: null } };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.getByText('Vuoto')).toBeInTheDocument();
  });

  it('shows noData text when row has no values', () => {
    const row: Row = { values: {} };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.queryByText('Dettaglio Record')).toBeInTheDocument();
  });

  it('renders title', () => {
    const row: Row = { values: { x: 'y' } };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.getByText('Dettaglio Record')).toBeInTheDocument();
  });

  it('renders close button', () => {
    const row: Row = { values: { a: 'b' } };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.getByRole('button', { name: 'Close detail panel' })).toBeInTheDocument();
  });

  it('calls onClose when close button clicked', async () => {
    const onClose = vi.fn();
    const row: Row = { values: { a: 'b' } };
    const { container } = render(<DetailPanel selectedRow={row} onClose={onClose} />);
    const btn = container.querySelector('button');
    btn?.click();
    expect(onClose).toHaveBeenCalled();
  });

  it('renders boolean values', () => {
    const row: Row = { values: { active: true } };
    render(<DetailPanel selectedRow={row} onClose={() => {}} />);
    expect(screen.getByText('true')).toBeInTheDocument();
  });
});
