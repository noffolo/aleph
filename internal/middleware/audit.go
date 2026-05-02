package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type AuditInterceptor struct {
	auditRepo *repository.AuditRepository
	logger    *slog.Logger
}

func NewAuditInterceptor(auditRepo *repository.AuditRepository, logger *slog.Logger) *AuditInterceptor {
	return &AuditInterceptor{
		auditRepo: auditRepo,
		logger:    logger,
	}
}

func (a *AuditInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		
		// Only audit after successful mutating operations
		if err == nil && isMutatingOperation(req.Spec().Procedure) {
			go a.logAuditEvent(ctx, req, resp)
		}
		
		return resp, err
	}
}

func (a *AuditInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a *AuditInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// Streaming audit not implemented yet
		return next(ctx, conn)
	}
}

func (a *AuditInterceptor) logAuditEvent(ctx context.Context, req connect.AnyRequest, resp connect.AnyResponse) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("audit panic", "recover", r)
		}
	}()
	procedure := req.Spec().Procedure
	projectID := ProjectIDFromContext(ctx)
	userID := "anonymous"
	if projectID != "" {
		userID = "project:" + projectID
	}

	action, resourceType, resourceID := extractAuditInfo(procedure, req, resp)
	if action == "" || resourceType == "" || resourceID == "" {
		return
	}

	entry := repository.AuditEntry{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ProjectID:    projectID,
		Timestamp:    time.Now(),
		Diff:         extractDiff(req, resp),
	}

	if err := a.auditRepo.InsertAuditLog(ctx, entry); err != nil {
		a.logger.Error("Failed to insert audit log", "error", err, "procedure", procedure)
	} else {
		a.logger.Info("Audit logged", "action", action, "resource", resourceType, "id", resourceID)
	}
}

func isMutatingOperation(procedure string) bool {
	// List of mutating procedures based on ConnectRPC service definitions
	// Exclude List, Get, Query, Search etc.
	switch {
	case strings.Contains(procedure, "Create"):
		return true
	case strings.Contains(procedure, "Update"):
		return true
	case strings.Contains(procedure, "Delete"):
		return true
	case strings.Contains(procedure, "Import"):
		return true
	case strings.Contains(procedure, "Export"):
		return true
	case strings.Contains(procedure, "Start"):
		return true
	case strings.Contains(procedure, "Stop"):
		return true
	case strings.Contains(procedure, "Execute"):
		return true
	case strings.Contains(procedure, "Run"):
		return true
	case strings.Contains(procedure, "Send"):
		return true
	default:
		return false
	}
}

func extractAuditInfo(procedure string, req connect.AnyRequest, resp connect.AnyResponse) (action, resourceType, resourceID string) {
	parts := strings.Split(procedure, ".")
	if len(parts) < 3 {
		return "", "", ""
	}
	
	methodName := parts[len(parts)-1]
	
	// Determine action from method name
	switch {
	case strings.HasPrefix(methodName, "Create"):
		action = "create"
	case strings.HasPrefix(methodName, "Update"):
		action = "update"
	case strings.HasPrefix(methodName, "Delete"):
		action = "delete"
	default:
		action = "modify"
	}
	
	// Determine resource type from service name
	serviceName := parts[len(parts)-2]
	switch {
	case strings.Contains(serviceName, "Agent"):
		resourceType = "agent"
	case strings.Contains(serviceName, "Tool"):
		resourceType = "tool"
	case strings.Contains(serviceName, "Skill"):
		resourceType = "skill"
	case strings.Contains(serviceName, "Ingestion"):
		resourceType = "ingestion"
	case strings.Contains(serviceName, "Task"):
		resourceType = "task"
	case strings.Contains(serviceName, "Notification"):
		resourceType = "notification"
	case strings.Contains(serviceName, "Auth"):
		resourceType = "api_key"
	case strings.Contains(serviceName, "Project"):
		resourceType = "project"
	default:
		resourceType = "unknown"
	}
	
	// Try to extract resource ID from request
	if req != nil {
		if reqMsg, ok := req.Any().(interface{ GetId() string }); ok {
			resourceID = reqMsg.GetId()
		}
		if resourceID == "" {
			if reqMsg, ok := req.Any().(interface{ GetResourceId() string }); ok {
				resourceID = reqMsg.GetResourceId()
			}
		}
	}
	if resourceID == "" {
		// Fallback: use timestamp
		resourceID = time.Now().Format("20060102150405")
	}

	return action, resourceType, resourceID
}

func extractDiff(req connect.AnyRequest, resp connect.AnyResponse) json.RawMessage {
	// For now, return empty diff
	// Could be enhanced to compare req/resp or store request body
	return nil
}