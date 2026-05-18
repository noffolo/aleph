package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/auth"
	"github.com/ff3300/aleph-v2/internal/repository"
)

var authSkipSet = map[string]bool{}

type AuthInterceptor struct {
	metaRepo        *repository.MetadataRepository
	jwtSecret       []byte
	revocationStore *TokenRevocationStore
}

func NewAuthInterceptor(metaRepo *repository.MetadataRepository, jwtSecret []byte) *AuthInterceptor {
	return &AuthInterceptor{
		metaRepo:        metaRepo,
		jwtSecret:       jwtSecret,
		revocationStore: NewTokenRevocationStore(1 * time.Hour),
	}
}

func NewAuthInterceptorWithRevocation(metaRepo *repository.MetadataRepository, jwtSecret []byte, revocationStore *TokenRevocationStore) *AuthInterceptor {
	return &AuthInterceptor{
		metaRepo:        metaRepo,
		jwtSecret:       jwtSecret,
		revocationStore: revocationStore,
	}
}

func (a *AuthInterceptor) RevocationStore() *TokenRevocationStore {
	return a.revocationStore
}

type procedureRole struct {
	adminOnly bool
	minRole   Role
}

var procedureRBAC = map[string]procedureRole{
	"/aleph.v1.AuthService/CreateApiKey": {adminOnly: true},
	"/aleph.v1.AuthService/ListApiKeys":  {adminOnly: true},
	"/aleph.v1.AuthService/DeleteApiKey": {adminOnly: true},

	"/aleph.v1.ProjectService/CreateProject":  {minRole: RoleUser},
	"/aleph.v1.ProjectService/DeleteProject":  {adminOnly: true},
	"/aleph.v1.ProjectService/ListProjects":   {},
	"/aleph.v1.ProjectService/EmergeOntology": {minRole: RoleUser},
	"/aleph.v1.ProjectService/GetOntology":    {},
	"/aleph.v1.ProjectService/SaveOntology":   {minRole: RoleUser},

	"/aleph.v1.AgentService/CreateAgent": {minRole: RoleUser},
	"/aleph.v1.AgentService/UpdateAgent": {minRole: RoleUser},
	"/aleph.v1.AgentService/DeleteAgent": {minRole: RoleUser},
	"/aleph.v1.AgentService/ListAgents":  {},
	"/aleph.v1.AgentService/ListModels":  {},

	"/aleph.v1.SkillService/ListSkills":  {},
	"/aleph.v1.SkillService/CreateSkill": {minRole: RoleUser},
	"/aleph.v1.SkillService/UpdateSkill": {minRole: RoleUser},
	"/aleph.v1.SkillService/DeleteSkill": {minRole: RoleUser},

	"/aleph.v1.ToolService/ListTools":  {},
	"/aleph.v1.ToolService/CreateTool": {minRole: RoleUser},
	"/aleph.v1.ToolService/UpdateTool": {minRole: RoleUser},
	"/aleph.v1.ToolService/DeleteTool": {minRole: RoleUser},

	"/aleph.v1.LibraryService/ListAssets":      {},
	"/aleph.v1.LibraryService/GetAssetContent": {},
	"/aleph.v1.LibraryService/DeleteAsset":     {minRole: RoleUser},
	"/aleph.v1.LibraryService/GeneratePdf":     {},
	"/aleph.v1.LibraryService/UploadAsset":     {minRole: RoleUser},

	"/aleph.v1.QueryService/ExecuteQuery":   {},
	"/aleph.v1.QueryService/Chat":           {},
	"/aleph.v1.QueryService/GetChatHistory": {},
	"/aleph.v1.QueryService/GetDataStats":   {},
	"/aleph.v1.QueryService/ConfirmAction":  {minRole: RoleUser},
	"/aleph.v1.QueryService/GlobalQuery":    {},
	"/aleph.v1.QueryService/GetDataLineage": {},
	"/aleph.v1.QueryService/GetChecksum":    {},

	"/aleph.v1.IngestionService/RunTask":     {minRole: RoleUser},
	"/aleph.v1.IngestionService/CreateTask":  {minRole: RoleUser},
	"/aleph.v1.IngestionService/DeleteTask":  {minRole: RoleUser},
	"/aleph.v1.IngestionService/ListTasks":   {},
	"/aleph.v1.IngestionService/GetTaskLogs": {},
	"/aleph.v1.IngestionService/GetProgress": {},

	"/aleph.v1.NotificationService/ListChannels": {},
	"/aleph.v1.NotificationService/SendWebhook":  {minRole: RoleUser},

	"/aleph.registry.v1.RegistryService/RegisterComponent":     {minRole: RoleUser},
	"/aleph.registry.v1.RegistryService/GetComponent":          {},
	"/aleph.registry.v1.RegistryService/ListComponents":        {},
	"/aleph.registry.v1.RegistryService/UpdateComponentStatus": {minRole: RoleUser},

	"/aleph.tool.v1.SandboxService/ExecuteTool": {minRole: RoleUser},
	"/aleph.tool.v1.SandboxService/RunSkill":    {minRole: RoleUser},

	"/aleph.nlp.v1.NLPService/AnalyzeSentiment": {minRole: RoleUser},
	"/aleph.nlp.v1.NLPService/ExtractEntities":  {minRole: RoleUser},
}

func checkProcedureRBAC(procedure string, role Role) error {
	rbac, ok := procedureRBAC[procedure]
	if !ok {
		return nil
	}
	if rbac.adminOnly && role != RoleAdmin {
		return ErrForbidden
	}
	if roleRank(role) < roleRank(rbac.minRole) {
		return ErrForbidden
	}
	return nil
}

func roleRank(r Role) int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleUser:
		return 2
	case RoleReadOnly:
		return 1
	default:
		return 0
	}
}

func (a *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if authSkipSet[req.Spec().Procedure] {
			return next(ctx, req)
		}

		projectID, role, err := a.authenticate(req.Header(), req.Peer().Query)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = projectIDToContext(ctx, projectID, role)

		if err := checkProcedureRBAC(req.Spec().Procedure, role); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}

		return next(ctx, req)
	}
}

func (a *AuthInterceptor) authenticate(header http.Header, query map[string][]string) (string, Role, error) {
	if tokenStr := jwtFromCookie(header); tokenStr != "" {
		return validateJWTWithRevocation(tokenStr, a.jwtSecret, a.revocationStore)
	}

	apiKey := extractApiKey(header)
	if apiKey != "" {
		return validateAPIKey(a.metaRepo, apiKey)
	}

	return "", "", ErrNoAPIKey
}

func (a *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if authSkipSet[conn.Spec().Procedure] {
			return next(ctx, conn)
		}

		projectID, role, err := a.authenticate(conn.RequestHeader(), nil)
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}

		ctx = projectIDToContext(ctx, projectID, role)

		if err := checkProcedureRBAC(conn.Spec().Procedure, role); err != nil {
			return connect.NewError(connect.CodePermissionDenied, err)
		}

		return next(ctx, conn)
	}
}

func extractApiKey(h http.Header) string {
	if key := h.Get("X-Aleph-Api-Key"); key != "" {
		return key
	}
	if authH := h.Get("Authorization"); authH != "" {
		return strings.TrimPrefix(authH, "Bearer ")
	}
	return ""
}

func jwtFromCookie(header http.Header) string {
	cookies := header.Values("Cookie")
	for _, cookie := range cookies {
		for _, c := range strings.Split(cookie, ";") {
			c = strings.TrimSpace(c)
			if after, found := strings.CutPrefix(c, "aleph_jwt="); found {
				return after
			}
		}
	}
	return ""
}

func validateJWT(tokenStr string, secret []byte) (string, Role, error) {
	claims, err := auth.ValidateToken(tokenStr, secret)
	if err != nil {
		return "", "", fmt.Errorf("invalid session: %w", err)
	}
	role := Role(claims.Role)
	if role == "" {
		role = RoleUser
	}
	return claims.ProjectID, role, nil
}

func validateJWTWithRevocation(tokenStr string, secret []byte, store *TokenRevocationStore) (string, Role, error) {
	claims, err := auth.ValidateToken(tokenStr, secret)
	if err != nil {
		return "", "", fmt.Errorf("invalid session: %w", err)
	}

	if store != nil && store.IsRevoked(claims.ID) {
		return "", "", fmt.Errorf("token has been revoked")
	}

	role := Role(claims.Role)
	if role == "" {
		role = RoleUser
	}
	return claims.ProjectID, role, nil
}

func validateAPIKey(metaRepo *repository.MetadataRepository, apiKey string) (string, Role, error) {
	return ValidateAPIKey(metaRepo, apiKey)
}

type TokenRevocationStore struct {
	mu      sync.RWMutex
	revoked map[string]time.Time
	ttl     time.Duration
	stopCh  chan struct{}
}

func NewTokenRevocationStore(ttl time.Duration) *TokenRevocationStore {
	s := &TokenRevocationStore{
		revoked: make(map[string]time.Time),
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func (s *TokenRevocationStore) Revoke(jti string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[jti] = time.Now()
}

func (s *TokenRevocationStore) IsRevoked(jti string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.revoked[jti]
	return ok
}

func (s *TokenRevocationStore) Stop() {
	close(s.stopCh)
}

func (s *TokenRevocationStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for jti, revokedAt := range s.revoked {
				if now.Sub(revokedAt) > s.ttl {
					delete(s.revoked, jti)
				}
			}
			s.mu.Unlock()
		}
	}
}

func ValidateScopes(required, tokenScopes string) bool {
	if required == "" {
		return true
	}
	scopeSet := map[string]bool{}
	for _, s := range strings.Split(tokenScopes, ",") {
		scopeSet[strings.TrimSpace(s)] = true
	}
	for _, r := range strings.Split(required, ",") {
		r = strings.TrimSpace(r)
		if r != "" && !scopeSet[r] {
			return false
		}
	}
	return true
}
