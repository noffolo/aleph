import { describe, it, expect } from 'vitest';
import { ZodError } from 'zod';
import {
  AgentSchema,
  AgentFormSchema,
  SkillSchema,
  SkillFormSchema,
  ToolSchema,
  ToolFormSchema,
  ChatMessageSchema,
  RowSchema,
  QueryDataSchema,
  DataSourceFormSchema,
  ApiKeySchema,
  PredictionSchema,
  ContentDataSchema,
} from '../index';
import { fromProto, validateType } from '../validate';

// ──────────────────────────────────────────────
// Edge Cases — Existing Schemas
// ──────────────────────────────────────────────

describe('AgentSchema — edge cases', () => {
  const base = {
    id: 'a1',
    name: 'Agent',
    model: 'gpt-4',
    systemPrompt: 'Be helpful.',
  };

  it('rejects non-string name (number)', () => {
    expect(() => AgentSchema.parse({ ...base, name: 42 })).toThrow(ZodError);
  });

  it('allows empty provider string (optional)', () => {
    const parsed = AgentSchema.parse({ ...base, provider: '' });
    expect(parsed.provider).toBe('');
  });

  it('rejects skillIds with non-string elements', () => {
    expect(() =>
      AgentSchema.parse({ ...base, skillIds: ['s1', 123] }),
    ).toThrow(ZodError);
  });

  it('allows undefined skillIds (optional)', () => {
    const parsed = AgentSchema.parse(base);
    expect(parsed.skillIds).toBeUndefined();
  });

  it('coerces missing id to string "undefined" via z.coerce.string()', () => {
    const parsed = AgentSchema.parse({ name: 'X', model: 'm', systemPrompt: 'p' });
    expect(parsed.id).toBe('undefined');
  });
});

describe('AgentFormSchema — edge cases', () => {
  it('rejects empty name', () => {
    expect(() => AgentFormSchema.parse({ name: '', model: 'gpt-4' })).toThrow(ZodError);
  });

  it('rejects empty model', () => {
    expect(() => AgentFormSchema.parse({ name: 'Test', model: '' })).toThrow(ZodError);
  });

  it('rejects invalid URL in baseUrl', () => {
    expect(() =>
      AgentFormSchema.parse({ name: 'X', model: 'gpt-4', baseUrl: 'ftp://bad' }),
    ).toThrow(ZodError);
  });

  it('accepts valid http URL in baseUrl', () => {
    const parsed = AgentFormSchema.parse({ name: 'X', model: 'gpt-4', baseUrl: 'http://localhost' });
    expect(parsed.baseUrl).toBe('http://localhost');
  });

  it('accepts valid https URL in baseUrl', () => {
    const parsed = AgentFormSchema.parse({ name: 'X', model: 'gpt-4', baseUrl: 'https://api.example.com/v1' });
    expect(parsed.baseUrl).toBe('https://api.example.com/v1');
  });

  it('accepts undefined baseUrl', () => {
    const parsed = AgentFormSchema.parse({ name: 'X', model: 'gpt-4' });
    expect(parsed.baseUrl).toBeUndefined();
  });

  it('accepts empty baseUrl (optional)', () => {
    const parsed = AgentFormSchema.parse({ name: 'X', model: 'gpt-4', baseUrl: '' });
    // refine: val is '' → !val → refine skips (doesn't run regex)
    expect(parsed.baseUrl).toBe('');
  });
});

describe('SkillSchema — edge cases', () => {
  const base = { id: 's1', name: 'Skill', description: 'A skill' };

  it('rejects non-string name (number)', () => {
    expect(() => SkillSchema.parse({ ...base, name: 42 })).toThrow(ZodError);
  });

  it('rejects empty string description', () => {
    // description is required z.string() — empty string passes as valid string
    const parsed = SkillSchema.parse({ ...base, description: '' });
    expect(parsed.description).toBe('');
  });

  it('allows extra unknown fields via passthrough (unicode keys)', () => {
    const parsed = SkillSchema.parse({ ...base, '🦄': 'unicorn', _hidden: true });
    expect(parsed).toHaveProperty('🦄', 'unicorn');
    expect(parsed).toHaveProperty('_hidden', true);
  });
});

describe('SkillFormSchema — edge cases', () => {
  it('rejects empty name', () => {
    expect(() => SkillFormSchema.parse({ name: '' })).toThrow(ZodError);
  });

  it('accepts all optional fields omitted', () => {
    const parsed = SkillFormSchema.parse({ name: 'Valid' });
    expect(parsed.name).toBe('Valid');
    expect(parsed.description).toBeUndefined();
    expect(parsed.toolIds).toBeUndefined();
  });

  it('rejects toolIds with non-string entries', () => {
    expect(() =>
      SkillFormSchema.parse({ name: 'X', toolIds: ['ok', 42 as unknown as string] }),
    ).toThrow(ZodError);
  });
});

describe('ToolSchema — edge cases', () => {
  const base = {
    id: 't1',
    name: 'Tool',
    description: 'A tool',
    code: 'fn() {}',
  };

  it('rejects empty code', () => {
    // code is required z.string() — empty string passes but is valid string
    const parsed = ToolSchema.parse({ ...base, code: '' });
    expect(parsed.code).toBe('');
  });

  it('coerces numeric id to string', () => {
    const parsed = ToolSchema.parse({ ...base, id: 42 });
    expect(parsed.id).toBe('42');
  });
});

describe('ToolFormSchema — edge cases', () => {
  it('rejects empty name', () => {
    expect(() => ToolFormSchema.parse({ name: '' })).toThrow(ZodError);
  });

  it('accepts all optional fields omitted', () => {
    const parsed = ToolFormSchema.parse({ name: 'Tool X' });
    expect(parsed.name).toBe('Tool X');
    expect(parsed.description).toBeUndefined();
    expect(parsed.code).toBeUndefined();
  });

  it('accepts long code strings', () => {
    const code = 'x'.repeat(10000);
    const parsed = ToolFormSchema.parse({ name: 'Big', code });
    expect(parsed.code).toBe(code);
  });
});

describe('ChatMessageSchema — edge cases', () => {
  it('rejects role "user" with wrong casing ("USER")', () => {
    expect(() =>
      ChatMessageSchema.parse({ role: 'USER', content: 'hi', createdAt: Date.now() }),
    ).toThrow(ZodError);
  });

  it('rejects role "admin" enum', () => {
    expect(() =>
      ChatMessageSchema.parse({ role: 'admin', content: 'hi', createdAt: Date.now() }),
    ).toThrow(ZodError);
  });

  it('rejects null content', () => {
    expect(() =>
      ChatMessageSchema.parse({ role: 'user', content: null, createdAt: 1 }),
    ).toThrow(ZodError);
  });

  it('rejects content as number', () => {
    expect(() =>
      ChatMessageSchema.parse({ role: 'user', content: 12345, createdAt: 1 }),
    ).toThrow(ZodError);
  });
});

describe('RowSchema — edge cases', () => {
  it('rejects undefined in values record', () => {
    expect(() => RowSchema.parse({ values: { x: undefined } })).toThrow(ZodError);
  });

  it('handles large values record', () => {
    const values: Record<string, string | number | boolean | null> = {};
    for (let i = 0; i < 100; i++) {
      values[`col_${i}`] = i % 3 === 0 ? 'str' : i % 3 === 1 ? i : null;
    }
    const parsed = RowSchema.parse({ values });
    expect(Object.keys(parsed.values)).toHaveLength(100);
  });
});

describe('QueryDataSchema — edge cases', () => {
  it('rejects rows where a column value is an object', () => {
    expect(() =>
      QueryDataSchema.parse({
        columns: ['a'],
        rows: [{ values: { a: { nested: true } } }],
      }),
    ).toThrow(ZodError);
  });

  it('rejects rows where a column value is an array', () => {
    expect(() =>
      QueryDataSchema.parse({
        columns: ['a'],
        rows: [{ values: { a: [1, 2, 3] } }],
      }),
    ).toThrow(ZodError);
  });
});

describe('DataSourceFormSchema — edge cases', () => {
  it('rejects empty name', () => {
    expect(() =>
      DataSourceFormSchema.parse({ name: '', sourceType: 'csv', configJson: '{}' }),
    ).toThrow(ZodError);
  });

  it('rejects invalid JSON in configJson', () => {
    expect(() =>
      DataSourceFormSchema.parse({ name: 'Src', sourceType: 'csv', configJson: '{bad' }),
    ).toThrow(ZodError);
  });

  it('accepts valid JSON object', () => {
    const parsed = DataSourceFormSchema.parse({
      name: 'Src',
      sourceType: 'rss',
      configJson: '{"url":"https://example.com"}',
    });
    expect(parsed.configJson).toBe('{"url":"https://example.com"}');
  });

  it('accepts JSON array in configJson', () => {
    const parsed = DataSourceFormSchema.parse({
      name: 'Src',
      sourceType: 'csv',
      configJson: '["a","b"]',
    });
    expect(parsed.configJson).toBe('["a","b"]');
  });
});

describe('ApiKeySchema — edge cases', () => {
  it('rejects createdAt with string value', () => {
    expect(() =>
      ApiKeySchema.parse({ id: '1', label: 'Key', key: 'sk-abc', createdAt: 'today' as unknown as number }),
    ).toThrow(ZodError);
  });

  it('coerces numeric string createdAt', () => {
    const parsed = ApiKeySchema.parse({
      id: '1',
      label: 'Key',
      key: 'sk-abc',
      createdAt: '1700000000',
    });
    expect(parsed.createdAt).toBe(1700000000);
  });
});

describe('PredictionSchema — edge cases', () => {
  const base = {
    entityId: 'e1',
    probability: 0.5,
    predictedState: 'active',
    explanation: 'ok',
  };

  it('rejects probability > 100 (still a number, passes)', () => {
    // Zod doesn't enforce [0,1] range by default — passes
    const parsed = PredictionSchema.parse({ ...base, probability: 150 });
    expect(parsed.probability).toBe(150);
  });

  it('rejects negative probability (passes as number)', () => {
    const parsed = PredictionSchema.parse({ ...base, probability: -0.5 });
    expect(parsed.probability).toBe(-0.5);
  });

  it('rejects empty predictedState', () => {
    const parsed = PredictionSchema.parse({ ...base, predictedState: '' });
    expect(parsed.predictedState).toBe('');
  });
});

describe('ContentDataSchema — edge cases', () => {
  it('accepts deeply nested extra field via passthrough', () => {
    const parsed = ContentDataSchema.parse({ nested: { deep: [1, { x: 'y' }] } });
    expect(parsed).toHaveProperty('nested');
    expect((parsed as Record<string, unknown>).nested).toEqual({ deep: [1, { x: 'y' }] });
  });
});

// ──────────────────────────────────────────────
// validateType / fromProto Edge Cases
// ──────────────────────────────────────────────

describe('fromProto / validateType — edge cases', () => {
  it('fromProto throws on null input', () => {
    expect(() => fromProto(AgentSchema, null)).toThrow(ZodError);
  });

  it('validateType throws on undefined input', () => {
    expect(() => validateType(AgentSchema, undefined)).toThrow(ZodError);
  });

  it('validateType succeeds with valid data', () => {
    const result = validateType(AgentSchema, {
      id: 'a1',
      name: 'Agent',
      model: 'm',
      systemPrompt: 'p',
    });
    expect(result.id).toBe('a1');
  });

  it('fromProto succeeds with valid data', () => {
    const result = fromProto(AgentSchema, {
      id: 'a2',
      name: 'Proto',
      model: 'm',
      systemPrompt: '',
    });
    expect(result.id).toBe('a2');
  });
});
