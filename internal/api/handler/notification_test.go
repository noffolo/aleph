package handler

import (
	"testing"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"

	"connectrpc.com/connect"
)

func TestNewNotificationHandler(t *testing.T) {
	h := NewNotificationHandler(nil, (*repository.MetadataRepository)(nil))
	assert.NotNil(t, h)
	assert.Nil(t, h.svc)
	assert.Nil(t, h.repo)
}

func TestNotificationHandler_SendWebhook_RequestFields(t *testing.T) {
	req := connect.NewRequest(&v1.SendWebhookRequest{
		Url:         "https://hooks.example.com/webhook",
		Secret:      "whsec-abc123",
		PayloadJson: `{"event":"test"}`,
	})
	assert.Equal(t, "https://hooks.example.com/webhook", req.Msg.Url)
	assert.Equal(t, "whsec-abc123", req.Msg.Secret)
	assert.Equal(t, `{"event":"test"}`, req.Msg.PayloadJson)
}

func TestNotificationHandler_SendWebhook_NoSecret(t *testing.T) {
	req := connect.NewRequest(&v1.SendWebhookRequest{
		Url:         "https://hooks.example.com/webhook",
		Secret:      "",
		PayloadJson: `{"event":"test"}`,
	})
	assert.Empty(t, req.Msg.Secret)
}

func TestNotificationHandler_ListChannels_RequestFields(t *testing.T) {
	req := connect.NewRequest(&v1.ListChannelsRequest{
		ProjectId: "proj-1",
	})
	assert.Equal(t, "proj-1", req.Msg.ProjectId)
}
