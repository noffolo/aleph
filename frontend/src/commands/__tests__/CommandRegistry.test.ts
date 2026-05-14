import { describe, it, expect } from 'vitest'
import {
  type UnifiedCommand,
  type CommandResult,
  COMMAND_REGISTRY,
  parseCommand,
  getTabCompletion,
  executeCommand,
  formatHelp,
} from '../CommandRegistry'

describe('parseCommand', () => {
  it('returns null for empty input', () => {
    expect(parseCommand('')).toBeNull()
  })

  it('returns null for whitespace-only input', () => {
    expect(parseCommand('   ')).toBeNull()
  })

  it('returns null for input not starting with /', () => {
    expect(parseCommand('hello world')).toBeNull()
  })

  it('returns null for plain text without slash', () => {
    expect(parseCommand('how do I list tools?')).toBeNull()
  })

  it('parses a bare slash command with no args', () => {
    const result = parseCommand('/help')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/help')
    expect(result!.args).toBe('')
  })

  it('parses a slash command with args', () => {
    const result = parseCommand('/explore my-project')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/explore')
    expect(result!.args).toBe('my-project')
  })

  it('parses a slash command with multiple args', () => {
    const result = parseCommand('/tool install https://example.com/tool')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/tool')
    expect(result!.args).toBe('install https://example.com/tool')
  })

  it('lowercases the command portion', () => {
    const result = parseCommand('/HeLp')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/help')
  })

  it('preserves argument casing', () => {
    const result = parseCommand('/explore MyProject')
    expect(result).not.toBeNull()
    expect(result!.args).toBe('MyProject')
  })

  it('trims leading/trailing whitespace', () => {
    const result = parseCommand('  /clear  ')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/clear')
  })

  it('handles extra spaces between command and args', () => {
    const result = parseCommand('/agent     my-agent-id')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/agent')
    expect(result!.args).toBe('my-agent-id')
  })

  it('handles command with only slash and no text', () => {
    const result = parseCommand('/')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/')
    expect(result!.args).toBe('')
  })

  it('handles /model command with args', () => {
    const result = parseCommand('/model gpt-4o')
    expect(result).not.toBeNull()
    expect(result!.command).toBe('/model')
    expect(result!.args).toBe('gpt-4o')
  })
})

describe('getTabCompletion', () => {
  it('returns empty array for non-slash input', () => {
    expect(getTabCompletion('hello')).toEqual([])
  })

  it('returns empty array for empty string', () => {
    expect(getTabCompletion('')).toEqual([])
  })

  it('returns all commands matching prefix', () => {
    const completions = getTabCompletion('/tool')
    expect(completions.length).toBeGreaterThan(0)
    expect(completions.every((c) => c.startsWith('/tool'))).toBe(true)
  })

  it('returns exact match when full command typed', () => {
    const completions = getTabCompletion('/help')
    expect(completions).toContain('/help')
  })

  it('returns multiple results for ambiguous prefix', () => {
    const completions = getTabCompletion('/tool')
    expect(completions.length).toBeGreaterThan(1)
  })

  it('returns empty array for unmatched slash command', () => {
    expect(getTabCompletion('/zzz')).toEqual([])
  })

  it('matches case-insensitively (lowercases input)', () => {
    const completions = getTabCompletion('/HELP')
    expect(completions).toContain('/help')
  })

  it('returns all commands for bare slash', () => {
    const completions = getTabCompletion('/')
    expect(completions.length).toBe(COMMAND_REGISTRY.length)
  })

  it('returns multi-word command completions', () => {
    const completions = getTabCompletion('/tool ')
    expect(completions).toContain('/tool install')
    expect(completions).toContain('/tool list')
    expect(completions).toContain('/tool health')
  })
})

describe('executeCommand', () => {
  it('returns PASS_TO_LLM for empty input', () => {
    const result = executeCommand('')
    expect(result.handled).toBe(false)
    expect(result.action).toBe('PASS_TO_LLM')
  })

  it('returns PASS_TO_LLM for non-slash input', () => {
    const result = executeCommand('hello world')
    expect(result.handled).toBe(false)
    expect(result.action).toBe('PASS_TO_LLM')
  })

  it('returns PASS_TO_LLM for unknown slash command', () => {
    const result = executeCommand('/this-does-not-exist')
    expect(result.handled).toBe(false)
    expect(result.action).toBe('PASS_TO_LLM')
  })

  it('returns PASS_TO_LLM for gibberish slash command', () => {
    const result = executeCommand('/xyz123%%%')
    expect(result.handled).toBe(false)
    expect(result.action).toBe('PASS_TO_LLM')
  })

  it('returns PASS_TO_LLM for whitespace-only input', () => {
    const result = executeCommand('   ')
    expect(result.handled).toBe(false)
    expect(result.action).toBe('PASS_TO_LLM')
  })

  // ── SWITCH_COPILOT action ──────────────────────────────────────────

  it('handles /help → SWITCH_COPILOT with help message', () => {
    const result = executeCommand('/help')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SWITCH_COPILOT')
    expect(result.message).toBeDefined()
    expect(result.message).toContain('Comandi disponibili')
  })

  // ── CLEAR_CHAT action ──────────────────────────────────────────────

  it('handles /clear → CLEAR_CHAT', () => {
    const result = executeCommand('/clear')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('CLEAR_CHAT')
  })

  // ── SHOW_INLINE action ─────────────────────────────────────────────

  it('handles /explore → SHOW_INLINE with target', () => {
    const result = executeCommand('/explore')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('explore')
  })

  it('handles /settings → SHOW_INLINE with target', () => {
    const result = executeCommand('/settings')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('settings')
  })

  it('handles /predict → SHOW_INLINE', () => {
    const result = executeCommand('/predict')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('predict')
  })

  it('handles /library → SHOW_INLINE', () => {
    const result = executeCommand('/library')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('library')
  })

  it('handles /health → SHOW_INLINE', () => {
    const result = executeCommand('/health')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('health')
  })

  it('handles /tools → SHOW_INLINE', () => {
    const result = executeCommand('/tools')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('tool')
  })

  it('handles /components → SHOW_INLINE', () => {
    const result = executeCommand('/components')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('component')
  })

  it('handles /ontology → SHOW_INLINE', () => {
    const result = executeCommand('/ontology')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('ontology')
  })

  it('handles /skills → SHOW_INLINE', () => {
    const result = executeCommand('/skills')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('skill')
  })

  it('handles /agents → SHOW_INLINE', () => {
    const result = executeCommand('/agents')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('agent')
  })

  it('handles /agent with arg → SHOW_INLINE', () => {
    const result = executeCommand('/agent my-agent')
    expect(result.handled).toBe(true)
    expect(result.target).toBe('agent')
    expect(result.args).toBe('my-agent')
  })

  it('handles /data with sub-command arg', () => {
    const result = executeCommand('/data add')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SHOW_INLINE')
    expect(result.target).toBe('data')
    expect(result.args).toBe('add')
  })

  // ── AGENT_COMMAND action ───────────────────────────────────────────

  it('handles /model → AGENT_COMMAND', () => {
    const result = executeCommand('/model')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('AGENT_COMMAND')
  })

  it('handles /model with model name → AGENT_COMMAND with args', () => {
    const result = executeCommand('/model gpt-4')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('AGENT_COMMAND')
    expect(result.args).toBe('gpt-4')
  })

  // ── Case insensitivity ─────────────────────────────────────────────

  it('handles /HELP case-insensitively', () => {
    const result = executeCommand('/HELP')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('SWITCH_COPILOT')
  })

  it('handles /CleAr case-insensitively', () => {
    const result = executeCommand('/CleAr')
    expect(result.handled).toBe(true)
    expect(result.action).toBe('CLEAR_CHAT')
  })

  // ── Arg sanitization ───────────────────────────────────────────────

  it('sanitizes control characters from args', () => {
    // eslint-disable-next-line no-control-regex
    const result = executeCommand('/explore test\x00data')
    expect(result.args).not.toContain('\x00')
    expect(result.args).toBe('testdata')
  })

  it('sanitizes null byte from args', () => {
    // eslint-disable-next-line no-control-regex
    const result = executeCommand('/explore hello\x00world')
    expect(result.args).toBe('helloworld')
  })

  it('sanitizes DEL character (0x7F) from args', () => {
    // eslint-disable-next-line no-control-regex
    const result = executeCommand('/explore hello\x7Fworld')
    expect(result.args).toBe('helloworld')
  })

  it('trims args after sanitization', () => {
    // eslint-disable-next-line no-control-regex
    const result = executeCommand('/explore \x00test\x00')
    expect(result.args).toBe('test')
  })

  it('truncates very long args to 256 characters', () => {
    const longArg = 'a'.repeat(500)
    const result = executeCommand(`/explore ${longArg}`)
    expect(result.args).toBeDefined()
    expect(result.args!.length).toBeLessThanOrEqual(256)
  })

  it('handles args exactly at 256 characters', () => {
    const exactArg = 'a'.repeat(256)
    const result = executeCommand(`/explore ${exactArg}`)
    expect(result.args).toBeDefined()
    expect(result.args!.length).toBe(256)
  })

  it('passes undefined args when command has no args', () => {
    const result = executeCommand('/help')
    expect(result.args).toBeUndefined()
  })
})

describe('formatHelp', () => {
  it('starts with "Comandi disponibili:"', () => {
    const help = formatHelp()
    expect(help.startsWith('Comandi disponibili:\n')).toBe(true)
  })

  it('includes every command from the registry', () => {
    const help = formatHelp()
    for (const cmd of COMMAND_REGISTRY) {
      expect(help).toContain(cmd.name)
      expect(help).toContain(cmd.description)
    }
  })

  it('each command line follows the format "name — description"', () => {
    const help = formatHelp()
    const lines = help.split('\n')
    for (const line of lines.slice(1)) {
      if (line.trim() === '') continue
      expect(line).toMatch(/^\/.+ — .+$/)
    }
  })

  it('is not empty', () => {
    const help = formatHelp()
    expect(help.length).toBeGreaterThan(0)
  })
})

describe('COMMAND_REGISTRY integrity', () => {
  it('contains exactly 20 commands', () => {
    expect(COMMAND_REGISTRY).toHaveLength(20)
  })

  it('all commands have unique names', () => {
    const names = COMMAND_REGISTRY.map((c) => c.name)
    const uniqueNames = new Set(names)
    expect(uniqueNames.size).toBe(names.length)
  })

  it('all commands have valid action types', () => {
    const validActions = ['SWITCH_COPILOT', 'SHOW_INLINE', 'CLEAR_CHAT', 'AGENT_COMMAND', 'SET_INPUT']
    for (const cmd of COMMAND_REGISTRY) {
      expect(validActions).toContain(cmd.action)
    }
  })

  it('all commands have descriptions', () => {
    for (const cmd of COMMAND_REGISTRY) {
      expect(cmd.description).toBeTruthy()
      expect(typeof cmd.description).toBe('string')
      expect(cmd.description.trim().length).toBeGreaterThan(0)
    }
  })

  it('commands with requiresConfirmation have the flag set to true or false', () => {
    for (const cmd of COMMAND_REGISTRY) {
      if (cmd.requiresConfirmation !== undefined) {
        expect(typeof cmd.requiresConfirmation).toBe('boolean')
      }
    }
  })

  it('all commands start with /', () => {
    for (const cmd of COMMAND_REGISTRY) {
      expect(cmd.name.startsWith('/')).toBe(true)
    }
  })

  it('all SINGLE-WORD commands are executable via executeCommand', () => {
    // Multi-word commands like '/tool list' cannot be matched by
    // parseCommand which splits on whitespace.
    const singleWordCmds = COMMAND_REGISTRY.filter((c) => !c.name.includes(' '))
    for (const cmd of singleWordCmds) {
      const result = executeCommand(cmd.name)
      expect(result.handled).toBe(true)
    }
  })

  it('has exactly 5 commands requiring confirmation', () => {
    const confirmCmds = COMMAND_REGISTRY.filter((c) => c.requiresConfirmation === true)
    expect(confirmCmds).toHaveLength(5)
  })

  it('confirmation commands include /agent, /skills, /model, /tool install, /tool diagnose', () => {
    const confirmCmds = COMMAND_REGISTRY.filter((c) => c.requiresConfirmation === true)
    const names = confirmCmds.map((c) => c.name)
    expect(names).toEqual(
      expect.arrayContaining(['/agent', '/tool install', '/tool diagnose', '/skills', '/model'])
    )
  })

  it('all SHOW_INLINE commands have a target', () => {
    const inlineCmds = COMMAND_REGISTRY.filter((c) => c.action === 'SHOW_INLINE')
    for (const cmd of inlineCmds) {
      expect(cmd.target).toBeTruthy()
    }
  })
})

describe('CommandResult interface', () => {
  it('PASS_TO_LLM result has handled=false', () => {
    const result = executeCommand('not a command')
    expect(result.handled).toBe(false)
    expect(result.action).toBe('PASS_TO_LLM')
  })

  it('handled commands have handled=true', () => {
    const result = executeCommand('/help')
    expect(result.handled).toBe(true)
  })

  it('SWITCH_COPILOT result has message field', () => {
    const result = executeCommand('/help')
    expect(result.message).toBeDefined()
    expect(typeof result.message).toBe('string')
  })
})
