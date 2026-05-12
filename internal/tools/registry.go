// Package tools provides a centralized ToolRegistry for registering, listing,
// and executing all tool categories (finance, OSINT, human-ecosystems).
// This replaces the scattered switch/case dispatch previously in handler code.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ParamDef describes a single parameter accepted by a tool.
type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ToolDefinition is the canonical descriptor for any tool in the system.
// The Execute function uses a normalized signature: params are always
// map[string]any; the return value is marshalled to JSON by the caller.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	Params      []ParamDef             `json:"params,omitempty"`
	Execute     func(ctx context.Context, params map[string]any) (any, error) `json:"-"`
}

// ToolRegistry is a concurrency-safe central registry of tool definitions.
// Tools are keyed by "category:name" to prevent collisions.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

// NewToolRegistry creates an empty ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
}

// Register adds a single tool definition. Returns an error if the
// category:name key already exists (duplicate registration).
func (r *ToolRegistry) Register(def ToolDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if def.Category == "" {
		return fmt.Errorf("tool category is required for %q", def.Name)
	}
	if def.Execute == nil {
		return fmt.Errorf("tool %q has nil Execute function", def.Name)
	}

	key := toolKey(def.Category, def.Name)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[key]; exists {
		return fmt.Errorf("duplicate tool registration: %q (category=%q)", def.Name, def.Category)
	}

	r.tools[key] = def
	slog.Debug("tool registered", "category", def.Category, "name", def.Name)
	return nil
}

// RegisterAll registers multiple tool definitions atomically.
// If any definition fails, none are registered.
func (r *ToolRegistry) RegisterAll(defs []ToolDefinition) error {
	// Pre-validate all definitions
	for _, def := range defs {
		if def.Name == "" {
			return fmt.Errorf("tool with empty name in batch registration")
		}
		if def.Category == "" {
			return fmt.Errorf("tool %q has empty category", def.Name)
		}
		if def.Execute == nil {
			return fmt.Errorf("tool %q has nil Execute", def.Name)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates (existing + within batch)
	existing := make(map[string]bool, len(r.tools))
	for k := range r.tools {
		existing[k] = true
	}
	for _, def := range defs {
		key := toolKey(def.Category, def.Name)
		if existing[key] {
			return fmt.Errorf("duplicate tool in batch: %q (category=%q)", def.Name, def.Category)
		}
		existing[key] = true
	}

	// Register all
	for _, def := range defs {
		key := toolKey(def.Category, def.Name)
		r.tools[key] = def
		slog.Debug("tool registered (batch)", "category", def.Category, "name", def.Name)
	}
	return nil
}

// List returns tool definitions for the given category.
// If category is empty, returns all tools.
func (r *ToolRegistry) List(category string) []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if category == "" {
		result := make([]ToolDefinition, 0, len(r.tools))
		for _, def := range r.tools {
			result = append(result, def)
		}
		return result
	}

	result := make([]ToolDefinition, 0)
	for _, def := range r.tools {
		if def.Category == category {
			result = append(result, def)
		}
	}
	return result
}

// Get returns a single tool definition by category and name.
func (r *ToolRegistry) Get(category, name string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.tools[toolKey(category, name)]
	return def, ok
}

// Execute runs a tool by category and name with the given context and parameters.
// The context is wrapped with a 30-second timeout for per-tool execution.
func (r *ToolRegistry) Execute(ctx context.Context, category, name string, params map[string]any) (any, error) {
	def, ok := r.Get(category, name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %q in category %q", name, category)
	}
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return def.Execute(execCtx, params)
}

// ExecuteContext runs a tool by category and name with the given context and parameters.
func (r *ToolRegistry) ExecuteContext(ctx context.Context, category, name string, params map[string]any) (any, error) {
	def, ok := r.Get(category, name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %q in category %q", name, category)
	}
	return def.Execute(ctx, params)
}

// Categories returns all distinct category names.
func (r *ToolRegistry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	for _, def := range r.tools {
		seen[def.Category] = true
	}
	result := make([]string, 0, len(seen))
	for cat := range seen {
		result = append(result, cat)
	}
	return result
}

// toolKey returns the internal map key for a tool.
func toolKey(category, name string) string {
	return category + ":" + name
}

// ---------------------------------------------------------------------------
// Wrapper constructors — these normalize tool implementations with different
// Execute signatures into the canonical ToolDefinition form. They live here
// so the registry package is the single import needed by callers.
// ---------------------------------------------------------------------------

// FinanceToolDef wraps a finance tool (Execute takes map[string]any) into a ToolDefinition.
func FinanceToolDef(name, description string, execute func(ctx context.Context, params map[string]any) (any, error)) ToolDefinition {
	return ToolDefinition{
		Name:        name,
		Category:    "finance",
		Description: description,
		Execute:     execute,
	}
}

// OSINTToolDef wraps an OSINT tool (Execute takes JSON string, returns JSON string)
// into a normalized ToolDefinition. The wrapper marshals params → JSON string for input
// and unmarshals the result JSON string → any for output.
func OSINTToolDef(name, description string, execute func(ctx context.Context, argsJSON string) (string, error)) ToolDefinition {
	return ToolDefinition{
		Name:        name,
		Category:    "osint",
		Description: description,
		Execute: func(ctx context.Context, params map[string]any) (any, error) {
			raw, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("osint marshal params: %w", err)
			}
			resultStr, err := execute(ctx, string(raw))
			if err != nil {
				return nil, err
			}
			// Attempt to parse the JSON string result into structured data
			var parsed any
			if err := json.Unmarshal([]byte(resultStr), &parsed); err == nil {
				return parsed, nil
			}
			return resultStr, nil
		},
	}
}

// HEToolDef wraps a human-ecosystems tool (implements the ToolExecutor interface
// with Execute(ctx, map[string]any) (any, error), Name(), Description()).
type HETool interface {
	Execute(ctx context.Context, args map[string]any) (any, error)
	Name() string
	Description() string
}

// HEToolDef converts a human-ecosystems ToolExecutor into a ToolDefinition.
func HEToolDef(tool HETool) ToolDefinition {
	return ToolDefinition{
		Name:        tool.Name(),
		Category:    "human-ecosystems",
		Description: tool.Description(),
		Execute:     tool.Execute,
	}
}


