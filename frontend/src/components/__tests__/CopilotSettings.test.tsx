import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CopilotSettings } from '../CopilotSettings';
import type { ChatMessage } from '../../store/types';

describe('CopilotSettings', () => {
  it('renders empty state when no message', () => {
    render(<CopilotSettings message={null} onClose={() => {}} />);
    expect(screen.getByText('Seleziona un messaggio per vedere i dettagli')).toBeInTheDocument();
  });

  it('renders message role', () => {
    const msg: ChatMessage = {
      role: 'assistant',
      content: 'Hello world',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText('assistant')).toBeInTheDocument();
  });

  it('renders message content', () => {
    const msg: ChatMessage = {
      role: 'user',
      content: 'What is the forecast?',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText('What is the forecast?')).toBeInTheDocument();
  });

  it('renders tool call when present', () => {
    const msg: ChatMessage = {
      role: 'assistant',
      content: 'Running analysis',
      toolCall: 'analyze_data',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText('Tool Call')).toBeInTheDocument();
    expect(screen.getByText('analyze_data')).toBeInTheDocument();
  });

  it('does not render tool call when absent', () => {
    const msg: ChatMessage = {
      role: 'assistant',
      content: 'Simple response',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.queryByText('Tool Call')).not.toBeInTheDocument();
  });

  it('renders role label', () => {
    const msg: ChatMessage = {
      role: 'system',
      content: 'System setup',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText('Ruolo')).toBeInTheDocument();
  });

  it('renders content label', () => {
    const msg: ChatMessage = {
      role: 'user',
      content: 'Test',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText('Contenuto')).toBeInTheDocument();
  });

  it('renders detail header', () => {
    const msg: ChatMessage = {
      role: 'assistant',
      content: 'Hi',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText('Dettagli Messaggio')).toBeInTheDocument();
  });

  it('renders timestamp from createdAt', () => {
    const msg: ChatMessage = {
      role: 'user',
      content: 'Hi',
      createdAt: 1700000000,
    };
    render(<CopilotSettings message={msg} onClose={() => {}} />);
    expect(screen.getByText(/2023/)).toBeInTheDocument();
  });
});
