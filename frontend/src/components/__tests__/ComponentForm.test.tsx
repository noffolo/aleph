import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ComponentForm } from '../ComponentForm';
import type { RegistryComponent } from '../../store/types';

describe('ComponentForm', () => {
  const onSave = vi.fn();
  const onCancel = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders in create mode', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByRole('heading', { name: 'Registra Componente' })).toBeInTheDocument();
  });

  it('renders in edit mode with pre-filled data', () => {
    const component: RegistryComponent = {
      id: 'c1', name: 'MyComp', description: 'A component',
      version: '1.0.0', type: 'tool', category: 'analytical',
      source: 'user', status: 'active', approvalStatus: 'approved',
    };
    render(<ComponentForm component={component} onSave={onSave} onCancel={onCancel} title="Modifica Componente" />);
    expect(screen.getByText('Modifica Componente')).toBeInTheDocument();
    expect(screen.getByDisplayValue('MyComp')).toBeInTheDocument();
    expect(screen.getByDisplayValue('A component')).toBeInTheDocument();
  });

  it('renders title from prop', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} title="My Title" />);
    expect(screen.getByText('My Title')).toBeInTheDocument();
  });

  it('renders name input', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Nome')).toBeInTheDocument();
  });

  it('renders type select', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Tipo')).toBeInTheDocument();
  });

  it('renders description textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Descrizione')).toBeInTheDocument();
  });

  it('renders category select', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Categoria')).toBeInTheDocument();
  });

  it('renders source select', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Sorgente')).toBeInTheDocument();
  });

  it('renders status select', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Stato')).toBeInTheDocument();
  });

  it('renders approval select', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Approvazione')).toBeInTheDocument();
  });

  it('renders config schema textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Schema Config (JSON)')).toBeInTheDocument();
  });

  it('renders execution command input', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Comando Esecuzione')).toBeInTheDocument();
  });

  it('renders dependencies textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Dependencies (JSON)')).toBeInTheDocument();
  });

  it('renders tool IDs textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Tool IDs (JSON)')).toBeInTheDocument();
  });

  it('renders input schema textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Schema Input (JSON)')).toBeInTheDocument();
  });

  it('renders output schema textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Schema Output (JSON)')).toBeInTheDocument();
  });

  it('renders prompt template textarea', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Prompt Template')).toBeInTheDocument();
  });

  it('renders cancel button', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByText('Annulla')).toBeInTheDocument();
  });

  it('calls onCancel on cancel click', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    await user.click(screen.getByText('Annulla'));
    expect(onCancel).toHaveBeenCalled();
  });

  it('calls onSave with form data on submit', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByLabelText('Nome'), 'MyComponent');
    await user.click(screen.getByRole('button', { name: 'Registra Componente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ name: 'MyComponent' }));
  });

  it('shows validation error for empty name', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    await user.click(screen.getByRole('button', { name: 'Registra Componente' }));
    expect(onSave).not.toHaveBeenCalled();
  });

  it('pre-fills form fields from component prop', () => {
    const component: RegistryComponent = {
      id: 'c1', name: 'Test', description: 'Desc',
      version: '1.0.0', type: 'agent', category: 'integration',
      source: 'imported', status: 'inactive', approvalStatus: 'rejected',
      configSchemaJson: '{"key":"val"}', executionCommand: 'run.sh',
      dependenciesJson: '["dep1"]', inputSchemaJson: '{"in":"data"}',
      outputSchemaJson: '{"out":"result"}', promptTemplate: 'You are...',
      toolIdsJson: '["t1"]',
    };
    render(<ComponentForm component={component} onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByDisplayValue('Test')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Desc')).toBeInTheDocument();
    expect(screen.getByDisplayValue('run.sh')).toBeInTheDocument();
    expect(screen.getByDisplayValue('You are...')).toBeInTheDocument();
    expect(screen.getByDisplayValue('{"key":"val"}')).toBeInTheDocument();
    expect(screen.getByDisplayValue('["dep1"]')).toBeInTheDocument();
    expect(screen.getByDisplayValue('{"in":"data"}')).toBeInTheDocument();
    expect(screen.getByDisplayValue('{"out":"result"}')).toBeInTheDocument();
    expect(screen.getByDisplayValue('["t1"]')).toBeInTheDocument();
  });

  it('shows edit mode submit button text', () => {
    const component: RegistryComponent = {
      id: 'c1', name: 'MyComp', description: 'A component',
      version: '1.0.0', type: 'tool', category: 'analytical',
      source: 'user', status: 'active', approvalStatus: 'approved',
    };
    render(<ComponentForm component={component} onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByRole('button', { name: 'Registra Componente' })).toBeInTheDocument();
  });

  it('shows edit mode heading with title prop', () => {
    const component: RegistryComponent = {
      id: 'c1', name: 'C', description: '', version: '1.0.0',
      type: 'tool', category: 'analytical', source: 'user',
      status: 'active', approvalStatus: 'approved',
    };
    render(<ComponentForm component={component} onSave={onSave} onCancel={onCancel} title="Personalizza" />);
    expect(screen.getByText('Personalizza')).toBeInTheDocument();
  });

  it('changes type select to tool', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Tipo');
    await user.selectOptions(select, 'tool');
    expect((select as HTMLSelectElement).value).toBe('tool');
  });

  it('changes category select to analytical', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Categoria');
    await user.selectOptions(select, 'analytical');
    expect((select as HTMLSelectElement).value).toBe('analytical');
  });

  it('changes source select to imported', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Sorgente');
    await user.selectOptions(select, 'imported');
    expect((select as HTMLSelectElement).value).toBe('imported');
  });

  it('changes status select to inactive', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Stato');
    await user.selectOptions(select, 'inactive');
    expect((select as HTMLSelectElement).value).toBe('inactive');
  });

  it('changes approval select to rejected', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Approvazione');
    await user.selectOptions(select, 'rejected');
    expect((select as HTMLSelectElement).value).toBe('rejected');
  });

  it('types in config schema textarea', async () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const textarea = screen.getByLabelText('Schema Config (JSON)') as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: '{"fields":[{"name":"x"}]}' } });
    expect(textarea.value).toBe('{"fields":[{"name":"x"}]}');
  });

  it('types in description textarea', async () => {
    const user = userEvent.setup();
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const textarea = screen.getByLabelText('Descrizione');
    await user.type(textarea, 'A useful component');
    expect(textarea).toHaveDisplayValue('A useful component');
  });

  it('submits edited component with all fields', async () => {
    const user = userEvent.setup();
    const component: RegistryComponent = {
      id: 'c1', name: 'Old', description: 'Old desc', version: '1.0.0',
      type: 'skill', category: 'generative', source: 'user',
      status: 'pending', approvalStatus: 'pending',
    };
    render(<ComponentForm component={component} onSave={onSave} onCancel={onCancel} />);
    await user.click(screen.getByRole('button', { name: 'Registra Componente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Old', description: 'Old desc', type: 'skill',
    }));
  });

  it('renders type select with all options', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Tipo');
    const options = Array.from((select as HTMLSelectElement).options).map(o => o.value);
    expect(options).toContain('skill');
    expect(options).toContain('tool');
    expect(options).toContain('agent');
    expect(options).toContain('model');
    expect(options).toContain('pipeline');
  });

  it('renders category select with all options', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Categoria');
    const options = Array.from((select as HTMLSelectElement).options).map(o => o.value);
    expect(options).toContain('generative');
    expect(options).toContain('analytical');
    expect(options).toContain('transformative');
    expect(options).toContain('integration');
    expect(options).toContain('orchestration');
  });

  it('renders approval select with all options', () => {
    render(<ComponentForm onSave={onSave} onCancel={onCancel} />);
    const select = screen.getByLabelText('Approvazione');
    const options = Array.from((select as HTMLSelectElement).options).map(o => o.value);
    expect(options).toContain('pending');
    expect(options).toContain('approved');
    expect(options).toContain('rejected');
    expect(options).toContain('review');
  });
});
