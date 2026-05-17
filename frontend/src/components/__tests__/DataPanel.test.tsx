import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DataPanel } from '../DataPanel';

describe('DataPanel', () => {
  it('returns null when not open', () => {
    const { container } = render(
      <DataPanel data={{}} isOpen={false} onClose={() => {}} />
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders when open', () => {
    const { container } = render(
      <DataPanel data={{ name: 'Test' }} isOpen={true} onClose={() => {}} />
    );
    expect(container.firstChild).toBeInTheDocument();
  });

  it('renders custom title', () => {
    render(
      <DataPanel data={{}} isOpen={true} onClose={() => {}} title="My Panel" />
    );
    expect(screen.getByText('My Panel')).toBeInTheDocument();
  });

  it('renders default title when not provided', () => {
    render(
      <DataPanel data={{}} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('DETAIL')).toBeInTheDocument();
  });

  it('renders data entries', () => {
    const data = { name: 'Alice', age: 30 };
    render(
      <DataPanel data={data} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('30')).toBeInTheDocument();
    expect(screen.getByText('name')).toBeInTheDocument();
    expect(screen.getByText('age')).toBeInTheDocument();
  });

  it('shows no data message when data is null', () => {
    render(
      <DataPanel data={null} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('No data selected')).toBeInTheDocument();
  });

  it('renders null values as em-dash', () => {
    const data = { empty: null };
    render(
      <DataPanel data={data} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('renders children', () => {
    render(
      <DataPanel data={{}} isOpen={true} onClose={() => {}}>
        <div data-testid="child">Extra Content</div>
      </DataPanel>
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });

  it('renders close button', () => {
    render(
      <DataPanel data={{}} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('ESC')).toBeInTheDocument();
  });

  it('renders boolean values', () => {
    const data = { active: true, visible: false };
    render(
      <DataPanel data={data} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('true')).toBeInTheDocument();
    expect(screen.getByText('false')).toBeInTheDocument();
  });

  it('renders numeric zero', () => {
    const data = { count: 0 };
    render(
      <DataPanel data={data} isOpen={true} onClose={() => {}} />
    );
    expect(screen.getByText('0')).toBeInTheDocument();
  });
});
