import { describe, it, expect } from "vitest";
import { ZodError } from "zod";
import {
  ApiKeySchema,
  ProjectSchema,
  NotificationChannelSchema,
  RegistryComponentSchema,
  ChatMessageSchema,
  PendingConfirmationSchema,
  AgentSchema,
  SkillSchema,
  ToolSchema,
  PredictionSchema,
  IngestionTaskSchema,
  RowSchema,
  QueryDataSchema,
  SandboxResultSchema,
  AssetSchema,
  ColumnStatsSchema,
  ContentDataSchema,
} from "../index";

// ──────────────────────────────────────────────
// ApiKeySchema
// ──────────────────────────────────────────────
describe("ApiKeySchema", () => {
  const valid = { id: "1", label: "My Key", key: "sk-abc123", createdAt: 1234567890 };

  it("validates a correct ApiKey", () => {
    const parsed = ApiKeySchema.parse(valid);
    expect(parsed.label).toBe("My Key");
    expect(parsed.key).toBe("sk-abc123");
  });

  it("rejects missing label", () => {
    expect(() => ApiKeySchema.parse({ id: "1", key: "sk-abc", createdAt: 1 })).toThrow(ZodError);
  });

  it("rejects wrong type for createdAt", () => {
    expect(() => ApiKeySchema.parse({ ...valid, createdAt: "not-a-number" })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// ProjectSchema
// ──────────────────────────────────────────────
describe("ProjectSchema", () => {
  const valid = { id: "p1", name: "Test Project" };

  it("validates a correct Project", () => {
    const parsed = ProjectSchema.parse(valid);
    expect(parsed.name).toBe("Test Project");
  });

  it("rejects missing name", () => {
    expect(() => ProjectSchema.parse({ id: "p1" })).toThrow(ZodError);
  });

  it("rejects non-string name", () => {
    expect(() => ProjectSchema.parse({ id: "p1", name: 123 })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// NotificationChannelSchema
// ──────────────────────────────────────────────
describe("NotificationChannelSchema", () => {
  const valid = { id: "n1", name: "Email", type: "email", configJson: '{"recipient":"admin@test.com"}' };

  it("validates a correct NotificationChannel", () => {
    const parsed = NotificationChannelSchema.parse(valid);
    expect(parsed.type).toBe("email");
    expect(parsed.configJson).toBe('{"recipient":"admin@test.com"}');
  });

  it("rejects missing configJson", () => {
    expect(() => NotificationChannelSchema.parse({ id: "n1", name: "Email", type: "email" })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// RegistryComponentSchema
// ──────────────────────────────────────────────
describe("RegistryComponentSchema", () => {
  const valid = {
    id: "rc1",
    name: "Test Component",
    description: "A test",
    version: "1.0.0",
    type: "agent",
    category: "nlp",
    source: "built-in",
    status: "active",
    approvalStatus: "approved",
  };

  it("validates a correct RegistryComponent", () => {
    const parsed = RegistryComponentSchema.parse(valid);
    expect(parsed.name).toBe("Test Component");
    expect(parsed.status).toBe("active");
  });

  it("rejects missing required field (version)", () => {
    expect(() => RegistryComponentSchema.parse({ ...valid, version: undefined })).toThrow(ZodError);
  });

  it("accepts optional fields when present", () => {
    const withOpts = {
      ...valid,
      trustScore: 0.95,
      avgLatencyMs: 120,
      promptTemplate: "You are {{role}}",
    };
    const parsed = RegistryComponentSchema.parse(withOpts);
    expect(parsed.trustScore).toBe(0.95);
    expect(parsed.avgLatencyMs).toBe(120);
    expect(parsed.promptTemplate).toBe("You are {{role}}");
  });

  it("allows optional fields to be omitted", () => {
    const parsed = RegistryComponentSchema.parse(valid);
    expect(parsed.trustScore).toBeUndefined();
    expect(parsed.avgLatencyMs).toBeUndefined();
  });
});

// ──────────────────────────────────────────────
// ChatMessageSchema
// ──────────────────────────────────────────────
describe("ChatMessageSchema", () => {
  const valid = { role: "user" as const, content: "Hello world", createdAt: 12345 };

  it("validates a correct ChatMessage", () => {
    const parsed = ChatMessageSchema.parse(valid);
    expect(parsed.role).toBe("user");
    expect(parsed.content).toBe("Hello world");
  });

  it("rejects invalid role enum value", () => {
    expect(() => ChatMessageSchema.parse({ role: "admin", content: "test", createdAt: 1 })).toThrow(ZodError);
  });

  it("validates all three valid roles", () => {
    for (const role of ["user", "assistant", "system"] as const) {
      const parsed = ChatMessageSchema.parse({ role, content: "test", createdAt: 1 });
      expect(parsed.role).toBe(role);
    }
  });

  it("accepts optional fields", () => {
    const withOpts = { ...valid, toolCall: "fn_call", requiresConfirmation: true };
    const parsed = ChatMessageSchema.parse(withOpts);
    expect(parsed.toolCall).toBe("fn_call");
    expect(parsed.requiresConfirmation).toBe(true);
  });
});

// ──────────────────────────────────────────────
// PendingConfirmationSchema
// ──────────────────────────────────────────────
describe("PendingConfirmationSchema", () => {
  const valid = { projectId: "p1", agentId: "a1" };

  it("validates a correct PendingConfirmation", () => {
    const parsed = PendingConfirmationSchema.parse(valid);
    expect(parsed.projectId).toBe("p1");
    expect(parsed.agentId).toBe("a1");
  });

  it("rejects missing agentId", () => {
    expect(() => PendingConfirmationSchema.parse({ projectId: "p1" })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// AgentSchema (passthrough)
// ──────────────────────────────────────────────
describe("AgentSchema", () => {
  const valid = {
    id: "a1",
    name: "Test Agent",
    model: "gpt-4",
    systemPrompt: "You are a helpful assistant",
  };

  it("validates a correct Agent", () => {
    const parsed = AgentSchema.parse(valid);
    expect(parsed.name).toBe("Test Agent");
    expect(parsed.model).toBe("gpt-4");
  });

  it("rejects missing required field (systemPrompt)", () => {
    expect(() => AgentSchema.parse({ id: "a1", name: "Test", model: "gpt-4" })).toThrow(ZodError);
  });

  it("allows passthrough extra fields", () => {
    const withExtra = { ...valid, extraField: "hello", anotherField: 42 };
    const parsed = AgentSchema.parse(withExtra);
    expect(parsed).toHaveProperty("extraField", "hello");
    expect(parsed).toHaveProperty("anotherField", 42);
  });

  it("accepts optional fields", () => {
    const withOpts = { ...valid, provider: "openai", apiKey: "sk-xxx", skillIds: ["s1", "s2"] };
    const parsed = AgentSchema.parse(withOpts);
    expect(parsed.provider).toBe("openai");
    expect(parsed.skillIds).toEqual(["s1", "s2"]);
  });
});

// ──────────────────────────────────────────────
// SkillSchema (passthrough)
// ──────────────────────────────────────────────
describe("SkillSchema", () => {
  const valid = { id: "s1", name: "Test Skill", description: "A test skill" };

  it("validates a correct Skill", () => {
    const parsed = SkillSchema.parse(valid);
    expect(parsed.name).toBe("Test Skill");
  });

  it("rejects missing description", () => {
    expect(() => SkillSchema.parse({ id: "s1", name: "Test" })).toThrow(ZodError);
  });

  it("allows passthrough extra fields", () => {
    const withExtra = { ...valid, category: "nlp", version: 2 };
    const parsed = SkillSchema.parse(withExtra);
    expect(parsed).toHaveProperty("category", "nlp");
    expect(parsed).toHaveProperty("version", 2);
  });

  it("accepts optional toolIds", () => {
    const parsed = SkillSchema.parse({ ...valid, toolIds: ["t1", "t2"] });
    expect(parsed.toolIds).toEqual(["t1", "t2"]);
  });
});

// ──────────────────────────────────────────────
// ToolSchema (passthrough)
// ──────────────────────────────────────────────
describe("ToolSchema", () => {
  const valid = { id: "t1", name: "Test Tool", description: "A test tool", code: "function foo() {}" };

  it("validates a correct Tool", () => {
    const parsed = ToolSchema.parse(valid);
    expect(parsed.name).toBe("Test Tool");
    expect(parsed.code).toBe("function foo() {}");
  });

  it("rejects missing code", () => {
    expect(() => ToolSchema.parse({ id: "t1", name: "Test", description: "desc" })).toThrow(ZodError);
  });

  it("allows passthrough extra fields", () => {
    const withExtra = { ...valid, version: "1.0", author: "me" };
    const parsed = ToolSchema.parse(withExtra);
    expect(parsed).toHaveProperty("version", "1.0");
    expect(parsed).toHaveProperty("author", "me");
  });
});

// ──────────────────────────────────────────────
// PredictionSchema
// ──────────────────────────────────────────────
describe("PredictionSchema", () => {
  const valid = { entityId: "e1", probability: 0.85, predictedState: "active", explanation: "High confidence" };

  it("validates a correct Prediction", () => {
    const parsed = PredictionSchema.parse(valid);
    expect(parsed.probability).toBe(0.85);
    expect(parsed.predictedState).toBe("active");
  });

  it("rejects missing probability", () => {
    expect(() => PredictionSchema.parse({ entityId: "e1", predictedState: "active", explanation: "x" })).toThrow(ZodError);
  });

  it("rejects non-number probability", () => {
    expect(() => PredictionSchema.parse({ ...valid, probability: "high" })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// IngestionTaskSchema
// ──────────────────────────────────────────────
describe("IngestionTaskSchema", () => {
  const valid = { id: "t1", name: "Ingest CSV", sourceType: "csv", status: "running", progress: 50 };

  it("validates a correct IngestionTask", () => {
    const parsed = IngestionTaskSchema.parse(valid);
    expect(parsed.name).toBe("Ingest CSV");
    expect(parsed.progress).toBe(50);
  });

  it("rejects missing status", () => {
    expect(() => IngestionTaskSchema.parse({ id: "t1", name: "Ingest", sourceType: "csv", progress: 0 })).toThrow(ZodError);
  });

  it("rejects non-number progress", () => {
    expect(() => IngestionTaskSchema.parse({ ...valid, progress: "fifty" })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// RowSchema
// ──────────────────────────────────────────────
describe("RowSchema", () => {
  it("validates a correct Row with mixed value types", () => {
    const parsed = RowSchema.parse({ values: { name: "Alice", age: 30, active: true, note: null } });
    expect(parsed.values.name).toBe("Alice");
    expect(parsed.values.age).toBe(30);
    expect(parsed.values.active).toBe(true);
    expect(parsed.values.note).toBeNull();
  });

  it("validates empty values record", () => {
    const parsed = RowSchema.parse({ values: {} });
    expect(parsed.values).toEqual({});
  });

  it("rejects invalid value type (object)", () => {
    expect(() => RowSchema.parse({ values: { data: {} } })).toThrow(ZodError);
  });

  it("rejects invalid value type (array)", () => {
    expect(() => RowSchema.parse({ values: { list: [1, 2] } })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// QueryDataSchema
// ──────────────────────────────────────────────
describe("QueryDataSchema", () => {
  it("validates QueryData with all fields", () => {
    const parsed = QueryDataSchema.parse({
      columns: ["name", "age"],
      rows: [{ values: { name: "Alice", age: 30 } }, { values: { name: "Bob", age: 25 } }],
      sql: "SELECT * FROM users",
    });
    expect(parsed.columns).toEqual(["name", "age"]);
    expect(parsed.rows).toHaveLength(2);
    expect(parsed.sql).toBe("SELECT * FROM users");
  });

  it("validates empty QueryData (all optional)", () => {
    const parsed = QueryDataSchema.parse({});
    expect(parsed.columns).toBeUndefined();
    expect(parsed.rows).toBeUndefined();
    expect(parsed.sql).toBeUndefined();
  });

  it("rejects invalid row structure", () => {
    expect(() =>
      QueryDataSchema.parse({
        rows: [{ values: { nested: { invalid: true } } }],
      }),
    ).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// SandboxResultSchema (passthrough)
// ──────────────────────────────────────────────
describe("SandboxResultSchema", () => {
  it("validates a complete SandboxResult", () => {
    const parsed = SandboxResultSchema.parse({
      exitCode: 0,
      stdout: "Hello",
      stderr: "",
      metricsJson: '{"cpu": 0.5}',
    });
    expect(parsed.exitCode).toBe(0);
    expect(parsed.stdout).toBe("Hello");
  });

  it("validates empty object (all optional)", () => {
    const parsed = SandboxResultSchema.parse({});
    expect(parsed.exitCode).toBeUndefined();
    expect(parsed.stdout).toBeUndefined();
  });

  it("allows passthrough extra fields", () => {
    const parsed = SandboxResultSchema.parse({ extraField: "survives", another: true });
    expect(parsed).toHaveProperty("extraField", "survives");
    expect(parsed).toHaveProperty("another", true);
  });
});

// ──────────────────────────────────────────────
// AssetSchema
// ──────────────────────────────────────────────
describe("AssetSchema", () => {
  const valid = { id: "a1", name: "Report.pdf", type: "document", createdAt: 1234567890 };

  it("validates a correct Asset", () => {
    const parsed = AssetSchema.parse(valid);
    expect(parsed.name).toBe("Report.pdf");
    expect(parsed.type).toBe("document");
  });

  it("rejects missing type", () => {
    expect(() => AssetSchema.parse({ id: "a1", name: "Report.pdf", createdAt: 1 })).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// ColumnStatsSchema (bigint | number)
// ──────────────────────────────────────────────
describe("ColumnStatsSchema", () => {
  it("validates with bigint values", () => {
    const parsed = ColumnStatsSchema.parse({
      columnName: "age",
      min: "20",
      max: "60",
      count: 100n,
      uniqueCount: 40n,
      topValues: { "30": 50n, "40": 30n },
    });
    expect(parsed.columnName).toBe("age");
    expect(parsed.count).toBe(100n);
    expect(parsed.uniqueCount).toBe(40n);
  });

  it("validates with number values", () => {
    const parsed = ColumnStatsSchema.parse({
      columnName: "age",
      min: "20",
      max: "60",
      count: 100,
      uniqueCount: 40,
      topValues: { "30": 50, "40": 30 },
    });
    expect(parsed.count).toBe(100);
    expect(parsed.uniqueCount).toBe(40);
  });

  it("rejects missing columnName", () => {
    expect(() =>
      ColumnStatsSchema.parse({
        min: "20",
        max: "60",
        count: 100n,
        uniqueCount: 40n,
        topValues: {},
      }),
    ).toThrow(ZodError);
  });

  it("rejects invalid count type (string)", () => {
    expect(() =>
      ColumnStatsSchema.parse({
        columnName: "age",
        min: "20",
        max: "60",
        count: "one-hundred",
        uniqueCount: 40n,
        topValues: {},
      }),
    ).toThrow(ZodError);
  });
});

// ──────────────────────────────────────────────
// ContentDataSchema (passthrough, all optional)
// ──────────────────────────────────────────────
describe("ContentDataSchema", () => {
  it("validates with some fields", () => {
    const parsed = ContentDataSchema.parse({
      name: "Test",
      description: "A test content",
      code: "console.log('hi')",
      toolIds: ["t1"],
    });
    expect(parsed.name).toBe("Test");
    expect(parsed.toolIds).toEqual(["t1"]);
  });

  it("validates empty object (all optional)", () => {
    const parsed = ContentDataSchema.parse({});
    expect(Object.keys(parsed)).toHaveLength(0);
  });

  it("allows passthrough extra fields", () => {
    const parsed = ContentDataSchema.parse({ arbitraryKey: "value", flag: true, count: 99 });
    expect(parsed).toHaveProperty("arbitraryKey", "value");
    expect(parsed).toHaveProperty("flag", true);
    expect(parsed).toHaveProperty("count", 99);
  });

  it("rejects invalid toolIds (non-array)", () => {
    expect(() => ContentDataSchema.parse({ toolIds: "not-an-array" })).toThrow(ZodError);
  });
});
