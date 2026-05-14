import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AgentForm } from '../AgentForm';

// i18n mock — returns Italian translations like real t() does
vi.mock('../../i18n', () => ({
  t: (key: string) => {
    const map: Record<string, string> = {
      'agents.create': 'Nuovo Agente',
      'agents.edit': 'Modifica Agente',
      'agents.form.name': 'Es: Analista Finanze',
      'agents.form.model': 'Es: gpt-4o-mini o llama3.2',
      'agents.form.apiKey': 'Inserisci solo per sovrascrivere la chiave esistente (facoltativo)',
      'agents.form.baseUrl': 'Es: https://api.openai.com/v1',
      'agents.form.systemPrompt': 'Definisci il ruolo dell\'agente',
      'confirmDialog.cancel': 'Annulla',
    };
    return map[key] ?? key;
  },
}));

const mockGetState = vi.fn(() => ({
  agents: { agents: [], isLoading: false },
  tools: { tools: [], isLoading: false },
}));

vi.mock('../../store/useStore', () => ({
  useStore: Object.assign(
    vi.fn((sel: (s: Record<string, unknown>) => unknown) => sel(mockGetState())),
    { subscribe: vi.fn(() => vi.fn()), getState: mockGetState }
  ),
}));

describe('AgentForm', () => {
  const onSave = vi.fn();
  const onCancel = vi.fn();

  beforeEach(() => { vi.clearAllMocks(); });

  it('renders in create mode with title', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} title="Nuovo Agente" />);
    expect(screen.getByRole('heading', { name: 'Nuovo Agente' })).toBeInTheDocument();
  });

  it('renders name input with placeholder', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByPlaceholderText('Es: Analista Finanze')).toBeInTheDocument();
  });

  it('renders in edit mode with pre-filled data', () => {
    const agent = { id: 'a1', name: 'My Agent', model: 'gpt-4o', provider: 'openai',
      apiKey: '••••1234', baseUrl: 'https://api.x.com', systemPrompt: 'Be helpful' };
    render(<AgentForm agent={agent} onSave={onSave} onCancel={onCancel} title="Modifica Agente" />);
    expect(screen.getByText('Modifica Agente')).toBeInTheDocument();
    expect(screen.getByDisplayValue('My Agent')).toBeInTheDocument();
    expect(screen.getByDisplayValue('gpt-4o')).toBeInTheDocument();
  });

  it('calls onCancel on cancel click', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.click(screen.getByText('Annulla'));
    expect(onCancel).toHaveBeenCalled();
  });

  it('calls onSave with form data on submit', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Test');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ name: 'Test' }));
  });

  it('renders provider select', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByText('Provider')).toBeInTheDocument()
  })

  it('renders model input with placeholder', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByPlaceholderText('Es: gpt-4o-mini o llama3.2')).toBeInTheDocument()
  })

  it('renders system prompt textarea', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByPlaceholderText("Definisci il ruolo dell'agente")).toBeInTheDocument()
  })

  it('toggles API key visibility', async () => {
    const user = userEvent.setup();
    const { container } = render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    const input = screen.getByPlaceholderText('Inserisci solo per sovrascrivere la chiave esistente (facoltativo)')
    expect(input).toHaveAttribute('type', 'password')
    const toggleBtn = input.parentElement!.querySelector('button')!
    await user.click(toggleBtn)
    expect(input).toHaveAttribute('type', 'text')
  })

  it('pre-fills baseUrl in edit mode', () => {
    const agent = { id: 'a1', name: 'Test', model: 'gpt-4o', provider: 'openai',
      apiKey: '', baseUrl: 'https://custom.api.com', systemPrompt: '' }
    render(<AgentForm agent={agent} onSave={onSave} onCancel={onCancel} />)
    expect(screen.getByDisplayValue('https://custom.api.com')).toBeInTheDocument()
  })
});
