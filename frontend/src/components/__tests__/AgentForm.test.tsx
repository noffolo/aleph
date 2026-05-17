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
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
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

  it('shows validation error for empty name', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    const submitBtn = screen.getByRole('button', { name: 'Nuovo Agente' });
    await user.click(submitBtn);
    expect(onSave).not.toHaveBeenCalled();
  })

  it('shows validation error for empty model', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Test');
    const modelInput = screen.getByPlaceholderText('Es: gpt-4o-mini o llama3.2');
    await user.clear(modelInput);
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).not.toHaveBeenCalled();
  })

  it('validates baseUrl format', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Test');
    const baseUrlInput = screen.getByPlaceholderText('Es: https://api.openai.com/v1');
    await user.type(baseUrlInput, 'not-a-url');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).not.toHaveBeenCalled();
  })

  it('allows valid baseUrl', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Test');
    await user.type(screen.getByPlaceholderText('Es: https://api.openai.com/v1'), 'https://valid.url.com');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).toHaveBeenCalled();
  })

  it('changes provider select value', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    const providerSelect = screen.getByLabelText('Provider');
    await user.selectOptions(providerSelect, 'anthropic');
    expect((providerSelect as HTMLSelectElement).value).toBe('anthropic');
  })

  it('types in system prompt textarea', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    const textarea = screen.getByPlaceholderText("Definisci il ruolo dell'agente");
    await user.type(textarea, 'You are a helpful assistant');
    expect(textarea).toHaveDisplayValue('You are a helpful assistant');
  })

  it('renders AgentFormSchema validation label for name', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByLabelText('Nome')).toBeInTheDocument();
  })

  it('renders Aggiorna Agente in edit mode submit', () => {
    const agent = { id: 'a1', name: 'X', model: 'gpt-4o', systemPrompt: '' };
    render(<AgentForm agent={agent} onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByRole('button', { name: 'Aggiorna Agente' })).toBeInTheDocument();
  })

  it('submits with all fields filled', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Full Agent');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Full Agent',
      provider: 'openai',
      model: 'gpt-4o-mini',
    }));
  })

  it('pre-fills systemPrompt in edit mode', () => {
    const agent = { id: 'a1', name: 'A', model: 'gpt-4o', systemPrompt: 'Be precise' };
    render(<AgentForm agent={agent} onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByDisplayValue('Be precise')).toBeInTheDocument();
  })

  it('renders model error border when validation fails', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Test');
    await user.clear(screen.getByPlaceholderText('Es: gpt-4o-mini o llama3.2'));
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    const modelInput = screen.getByPlaceholderText('Es: gpt-4o-mini o llama3.2');
    expect(modelInput.className).toContain('border-danger');
  })

  it('renders baseUrl error border when validation fails', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Test');
    await user.type(screen.getByPlaceholderText('Es: https://api.openai.com/v1'), 'not-a-url');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    const baseUrlInput = screen.getByPlaceholderText('Es: https://api.openai.com/v1');
    expect(baseUrlInput.className).toContain('border-danger');
  })

  it('clears errors on successful re-submission', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).not.toHaveBeenCalled();
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Fixed');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).toHaveBeenCalled();
  })

  it('submits apiKey when filled', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Agent');
    const apiKeyInput = screen.getByPlaceholderText('Inserisci solo per sovrascrivere la chiave esistente (facoltativo)');
    const toggleBtn = apiKeyInput.parentElement!.querySelector('button')!;
    await user.click(toggleBtn);
    await user.type(apiKeyInput, 'sk-test-key');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ apiKey: 'sk-test-key' }));
  })

  it('submits systemPrompt when filled', async () => {
    const user = userEvent.setup();
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    await user.type(screen.getByPlaceholderText('Es: Analista Finanze'), 'Bot');
    await user.type(screen.getByPlaceholderText("Definisci il ruolo dell'agente"), 'Be concise');
    await user.click(screen.getByRole('button', { name: 'Nuovo Agente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ systemPrompt: 'Be concise' }));
  })

  it('submits all fields in edit mode', async () => {
    const user = userEvent.setup();
    const agent = { id: 'a1', name: 'Bot', model: 'gpt-4o', provider: 'anthropic',
      apiKey: 'sk-old', baseUrl: 'https://x.com', systemPrompt: 'Old prompt' };
    render(<AgentForm agent={agent} onSave={onSave} onCancel={onCancel} />);
    await user.click(screen.getByRole('button', { name: 'Aggiorna Agente' }));
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Bot', model: 'gpt-4o', provider: 'anthropic', systemPrompt: 'Old prompt',
    }));
  })

  it('shows correct heading in create mode without title prop', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByRole('heading', { name: 'Nuovo Agente' })).toBeInTheDocument();
  })

  it('shows correct heading in edit mode without title prop', () => {
    const agent = { id: 'a1', name: 'X', model: 'gpt-4o', systemPrompt: '' };
    render(<AgentForm agent={agent} onSave={onSave} onCancel={onCancel} />);
    expect(screen.getByRole('heading', { name: 'Modifica Agente' })).toBeInTheDocument();
  })

  it('renders API key as masked password by default', () => {
    render(<AgentForm onSave={onSave} onCancel={onCancel} />);
    const input = screen.getByPlaceholderText('Inserisci solo per sovrascrivere la chiave esistente (facoltativo)');
    expect(input).toHaveAttribute('type', 'password');
  })
});
