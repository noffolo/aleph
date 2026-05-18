package middleware

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/errors"
)

// SubsystemInterceptor annotates context with subsystem/operation metadata
// derived from the ConnectRPC procedure path, so the ErrorHandlerInterceptor
// can populate APIError fields.
type SubsystemInterceptor struct{}

func NewSubsystemInterceptor() *SubsystemInterceptor {
	return &SubsystemInterceptor{}
}

// subsystems maps procedure-path prefixes to subsystem names.
var procedureSubsystems = map[string]string{
	"aleph.v1.QueryService/":             "handler",
	"aleph.v1.ProjectService/":           "handler",
	"aleph.v1.AgentService/":             "handler",
	"aleph.v1.SkillService/":             "handler",
	"aleph.v1.ToolService/":              "handler",
	"aleph.v1.LibraryService/":           "handler",
	"aleph.v1.NotificationService/":      "handler",
	"aleph.v1.AuthService/":              "handler",
	"aleph.v1.IngestionService/":         "ingestion",
	"aleph.v1.SandboxService/":           "sandbox",
	"aleph.registry.v1.RegistryService/": "handler",
	"aleph.nlp.v1.NLPService/":           "nlp",
}

// operations maps procedure-name suffixes to operation names.
var procedureOperations = map[string]string{
	"ExecuteQuery":           "query",
	"GetChatHistory":         "query",
	"ConfirmAction":          "execute",
	"HealthCheck":            "health",
	"CreateProject":          "insert",
	"DeleteProject":          "delete",
	"GetProject":             "query",
	"ListProjects":           "query",
	"CreateAgent":            "insert",
	"UpdateAgent":            "update",
	"DeleteAgent":            "delete",
	"GetAgent":               "query",
	"ListAgents":             "query",
	"CreateSkill":            "insert",
	"UpdateSkill":            "update",
	"DeleteSkill":            "delete",
	"GetSkill":               "query",
	"ListSkills":             "query",
	"InstallTool":            "insert",
	"UninstallTool":          "delete",
	"GetTool":                "query",
	"ListTools":              "query",
	"GetToolHealth":          "health",
	"ExecuteTool":            "execute",
	"Execute":                "execute",
	"ListComponents":         "query",
	"GetComponent":           "query",
	"RegisterComponent":      "insert",
	"UpdateComponentStatus":  "update",
	"IngestFromRSS":          "insert",
	"IngestFromGitHub":       "insert",
	"IngestFromFile":         "insert",
	"IngestFromSitemap":      "insert",
	"IngestFromGoogleSheets": "insert",
	"IngestFromEmail":        "insert",
	"IngestFromURL":          "insert",
	"GetIngestionStatus":     "query",
	"ListIngestions":         "query",
	"StreamPredictions":      "execute",
	"RecordFeedback":         "insert",
	"AnalyzeSentiment":       "execute",
}

// deriveSubsystem extracts subsystem and operation from a ConnectRPC procedure path.
// Example: "/aleph.v1.QueryService/ExecuteQuery" → ("handler", "query")
func deriveSubsystem(procedure string) (subsystem, operation string) {
	path := strings.TrimPrefix(procedure, "/")

	// Match procedure prefix to subsystem
	for prefix, sub := range procedureSubsystems {
		if strings.HasPrefix(path, prefix) {
			subsystem = sub
			break
		}
	}

	// Extract operation from the last segment after "/"
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		op := path[idx+1:]
		if mapped, ok := procedureOperations[op]; ok {
			operation = mapped
		} else {
			operation = strings.ToLower(op)
		}
	}

	return
}

func (i *SubsystemInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		subsystem, operation := deriveSubsystem(req.Spec().Procedure)
		if subsystem != "" {
			ctx = errors.WithSubsystem(ctx, subsystem, operation)
		}
		return next(ctx, req)
	}
}

func (i *SubsystemInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *SubsystemInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		subsystem, operation := deriveSubsystem(conn.Spec().Procedure)
		if subsystem != "" {
			ctx = errors.WithSubsystem(ctx, subsystem, operation)
		}
		return next(ctx, conn)
	}
}
