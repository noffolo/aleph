export interface SlashCommand {
  name: string
  description: string
  usage?: string
  action: 'SWITCH_COPILOT' | 'SHOW_INLINE' | 'CLEAR_CHAT' | 'AGENT_COMMAND' | 'SET_INPUT'
  target?: string
  requiresConfirmation?: boolean
}

export const SLASH_COMMANDS: SlashCommand[] = [
  { name: '/help', description: 'Show available commands', usage: '/help', action: 'SWITCH_COPILOT' },
  { name: '/explore', description: 'Explore data and objects', usage: '/explore [object]', action: 'SHOW_INLINE', target: 'explore' },
  { name: '/agent', description: 'Interact with agents', usage: '/agent [id]', action: 'SHOW_INLINE', target: 'agent', requiresConfirmation: true },
  { name: '/ontology', description: 'View ontology graph', usage: '/ontology', action: 'SHOW_INLINE', target: 'ontology' },
  { name: '/data', description: 'View data sources', usage: '/data [add]', action: 'SHOW_INLINE', target: 'data' },
  { name: '/predict', description: 'View predictions', usage: '/predict', action: 'SHOW_INLINE', target: 'predict' },
  { name: '/library', description: 'View asset library', usage: '/library', action: 'SHOW_INLINE', target: 'library' },
  { name: '/health', description: 'View data health', usage: '/health', action: 'SHOW_INLINE', target: 'health' },
  { name: '/skills', description: 'View skills panel', usage: '/skills', action: 'SHOW_INLINE', target: 'skill', requiresConfirmation: true },
  { name: '/tools', description: 'View tools panel', usage: '/tools', action: 'SHOW_INLINE', target: 'tool' },
  { name: '/components', description: 'View registry components', usage: '/components', action: 'SHOW_INLINE', target: 'component' },
  { name: '/settings', description: 'Open settings', usage: '/settings', action: 'SHOW_INLINE', target: 'settings' },
  { name: '/clear', description: 'Clear chat history', usage: '/clear', action: 'CLEAR_CHAT' },
  { name: '/model', description: 'Show or change agent model', usage: '/model [name]', action: 'AGENT_COMMAND', requiresConfirmation: true },
]

export interface CommandResult {
  handled: boolean
  action: 'SWITCH_COPILOT' | 'SHOW_INLINE' | 'CLEAR_CHAT' | 'AGENT_COMMAND' | 'SET_INPUT' | 'PASS_TO_LLM'
  target?: string
  args?: string
  message?: string
}

export function parseCommand(input: string): { command: string; args: string } | null {
  const trimmed = input.trim()
  if (!trimmed.startsWith('/')) return null
  const parts = trimmed.split(/\s+/)
  return { command: parts[0].toLowerCase(), args: parts.slice(1).join(' ') }
}

export function getTabCompletion(input: string): string[] {
  if (!input.startsWith('/')) return []
  const partial = input.toLowerCase()
  return SLASH_COMMANDS.filter(c => c.name.startsWith(partial)).map(c => c.name)
}

const MAX_ARGS_LENGTH = 256
const SANITIZE_REGEX = /[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g

function sanitizeArgs(args: string): string {
  return args.replace(SANITIZE_REGEX, '').slice(0, MAX_ARGS_LENGTH).trim()
}

export function executeCommand(input: string): CommandResult {
  const parsed = parseCommand(input)
  if (!parsed) return { handled: false, action: 'PASS_TO_LLM' }

  const commandDef = SLASH_COMMANDS.find(c => c.name === parsed.command)
  if (!commandDef) return { handled: false, action: 'PASS_TO_LLM' }

  const safeArgs = parsed.args ? sanitizeArgs(parsed.args) : undefined

  switch (commandDef.action) {
    case 'CLEAR_CHAT':
      return { handled: true, action: 'CLEAR_CHAT' }
    case 'SHOW_INLINE':
      return { handled: true, action: 'SHOW_INLINE', target: commandDef.target, args: safeArgs }
    case 'AGENT_COMMAND':
      return { handled: true, action: 'AGENT_COMMAND', args: safeArgs }
    case 'SET_INPUT':
      return { handled: true, action: 'SET_INPUT', args: safeArgs }
    case 'SWITCH_COPILOT':
    default:
      return { handled: true, action: 'SWITCH_COPILOT', message: formatHelp() }
  }
}

function formatHelp(): string {
  return 'Comandi disponibili:\n' + SLASH_COMMANDS.map(c => `${c.name} — ${c.description}`).join('\n')
}
