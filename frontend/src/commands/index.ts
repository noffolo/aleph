export { COMMAND_REGISTRY } from './CommandRegistry'
export { parseCommand, getTabCompletion, executeCommand, formatHelp } from './CommandRegistry'
export type { UnifiedCommand, CommandResult } from './CommandRegistry'
// Backward-compat aliases
export type { UnifiedCommand as SlashCommand, CommandResult as SlashCommandResult } from './CommandRegistry'
