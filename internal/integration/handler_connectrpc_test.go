//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_AgentHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	createReq := &v1.CreateAgentRequest{ProjectId: "test-proj", Agent: &v1.Agent{Name: "test-agent", Provider: "openai", Model: "gpt-4o"}}
	raw, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.AgentService/CreateAgent", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	var createResp v1.CreateAgentResponse
	json.Unmarshal([]byte(readBody(t, resp)), &createResp)

	listReq := &v1.ListAgentsRequest{ProjectId: "test-proj"}
	raw, _ = json.Marshal(listReq)
	resp = ts.postConnectRPC("/aleph.v1.AgentService/ListAgents", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	assert.Contains(t, readBody(t, resp), "test-agent")

	deleteReq := &v1.DeleteAgentRequest{ProjectId: "test-proj", Id: createResp.Agent.Id}
	raw, _ = json.Marshal(deleteReq)
	resp = ts.postConnectRPC("/aleph.v1.AgentService/DeleteAgent", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
}

func TestIntegration_SkillHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	createReq := &v1.CreateSkillRequest{ProjectId: "test-proj", Skill: &v1.Skill{Name: "test-skill", Description: "A test skill"}}
	raw, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.SkillService/CreateSkill", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	var createResp v1.CreateSkillResponse
	json.Unmarshal([]byte(readBody(t, resp)), &createResp)

	listReq := &v1.ListSkillsRequest{ProjectId: "test-proj"}
	raw, _ = json.Marshal(listReq)
	resp = ts.postConnectRPC("/aleph.v1.SkillService/ListSkills", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	assert.Contains(t, readBody(t, resp), "test-skill")

	deleteReq := &v1.DeleteSkillRequest{ProjectId: "test-proj", Id: createResp.Skill.Id}
	raw, _ = json.Marshal(deleteReq)
	resp = ts.postConnectRPC("/aleph.v1.SkillService/DeleteSkill", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
}

func TestIntegration_ToolHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	createReq := &v1.CreateToolRequest{ProjectId: "test-proj", Tool: &v1.Tool{Name: "test-tool", Code: "func(){ return 42 }"}}
	raw, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.ToolService/CreateTool", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	var createResp v1.CreateToolResponse
	json.Unmarshal([]byte(readBody(t, resp)), &createResp)

	listReq := &v1.ListToolsRequest{ProjectId: "test-proj"}
	raw, _ = json.Marshal(listReq)
	resp = ts.postConnectRPC("/aleph.v1.ToolService/ListTools", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	assert.Contains(t, readBody(t, resp), "test-tool")

	deleteReq := &v1.DeleteToolRequest{ProjectId: "test-proj", Id: createResp.Tool.Id}
	raw, _ = json.Marshal(deleteReq)
	resp = ts.postConnectRPC("/aleph.v1.ToolService/DeleteTool", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
}

func TestIntegration_AuthHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	createReq := &v1.CreateApiKeyRequest{ProjectId: "test-proj", Label: "test-key"}
	raw, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.AuthService/CreateApiKey", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	var createResp v1.CreateApiKeyResponse
	json.Unmarshal([]byte(readBody(t, resp)), &createResp)

	listReq := &v1.ListApiKeysRequest{ProjectId: "test-proj"}
	raw, _ = json.Marshal(listReq)
	resp = ts.postConnectRPC("/aleph.v1.AuthService/ListApiKeys", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	assert.Contains(t, readBody(t, resp), "test-key")

	deleteReq := &v1.DeleteApiKeyRequest{ProjectId: "test-proj", Id: createResp.Key.Id}
	raw, _ = json.Marshal(deleteReq)
	resp = ts.postConnectRPC("/aleph.v1.AuthService/DeleteApiKey", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
}

func TestIntegration_ProjectHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	createReq := &v1.CreateProjectRequest{Id: "test-project", Name: "test-project"}
	raw, _ := json.Marshal(createReq)
	resp := ts.postConnectRPC("/aleph.v1.ProjectService/CreateProject", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	listReq := &v1.ListProjectsRequest{}
	raw, _ = json.Marshal(listReq)
	resp = ts.postConnectRPC("/aleph.v1.ProjectService/ListProjects", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	assert.Contains(t, readBody(t, resp), "test-project")
}

func TestIntegration_NotificationHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	listReq := &v1.ListChannelsRequest{ProjectId: "test-proj"}
	raw, _ := json.Marshal(listReq)
	resp := ts.postConnectRPC("/aleph.v1.NotificationService/ListChannels", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
}

func TestIntegration_RegistryHandlers_ConnectRPC(t *testing.T) {
	ts := newTestServer(t)
	jwtCookie := ts.loginAsAdmin(t)

	registerReq := &v1.RegisterComponentRequest{
		Metadata: &v1.ComponentMetadata{Name: "test-comp", Type: "tool", Version: "1.0"},
	}
	raw, _ := json.Marshal(registerReq)
	resp := ts.postConnectRPC("/aleph.v1.RegistryService/RegisterComponent", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)

	listReq := &v1.ListComponentsRequest{}
	raw, _ = json.Marshal(listReq)
	resp = ts.postConnectRPC("/aleph.v1.RegistryService/ListComponents", raw, []*http.Cookie{jwtCookie})
	requireStatus(t, resp, http.StatusOK)
	assert.Contains(t, readBody(t, resp), "test-comp")
}
