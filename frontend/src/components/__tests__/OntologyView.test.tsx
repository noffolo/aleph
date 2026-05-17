import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { OntologyView } from '../OntologyView';

vi.mock('lucide-react', () => ({
  Zap: () => <span data-testid="icon-zap">Zap</span>,
  Save: () => <span data-testid="icon-save">Save</span>,
  Code: () => <span data-testid="icon-code">Code</span>,
}));

describe('OntologyView', () => {
  it('renders loading state', () => {
    const { container } = render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} isLoading={true} />
    );
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
  });

  it('renders error state', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} error="Something went wrong" />
    );
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('renders main heading', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('Modellazione Business')).toBeInTheDocument();
  });

  it('renders Emerge button', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByRole('button', { name: 'Emerge automatic ontology' })).toBeInTheDocument();
  });

  it('renders Publish button', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByRole('button', { name: 'Publish ontology model' })).toBeInTheDocument();
  });

  it('renders textarea with ontologyRaw value', () => {
    const raw = 'object Test\n  property name';
    render(
      <OntologyView ontologyRaw={raw} setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    const textarea = screen.getByPlaceholderText(/Inizia a definire/);
    expect(textarea).toHaveDisplayValue(raw);
  });

  it('renders DSL editor label', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('Editor Codice DSL Aleph')).toBeInTheDocument();
  });

  it('renders Glossario Visivo section', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('Glossario Visivo')).toBeInTheDocument();
  });

  it('parses object blocks from ontologyRaw', () => {
    const raw = 'object Appalto\n  property bandi_attivi';
    render(
      <OntologyView ontologyRaw={raw} setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('Appalto')).toBeInTheDocument();
    expect(screen.getByText('object')).toBeInTheDocument();
  });

  it('parses enum blocks', () => {
    const raw = 'enum StatoAppalto\n  value aperto';
    render(
      <OntologyView ontologyRaw={raw} setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('StatoAppalto')).toBeInTheDocument();
    expect(screen.getByText('enum')).toBeInTheDocument();
  });

  it('parses relations in blocks', () => {
    const raw = 'object Appalto\n  property bandi\n  relation EnteAppaltante';
    render(
      <OntologyView ontologyRaw={raw} setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('Appalto')).toBeInTheDocument();
  });

  it('renders Tips section', () => {
    render(
      <OntologyView ontologyRaw="" setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('Tips')).toBeInTheDocument();
  });

  it('renders multiple blocks', () => {
    const raw = 'object A\n  property x\nobject B\n  property y\nenum C\n  value v';
    render(
      <OntologyView ontologyRaw={raw} setOntologyRaw={() => {}} onEmerge={() => {}} onSave={() => {}} />
    );
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
    expect(screen.getByText('C')).toBeInTheDocument();
  });
});
