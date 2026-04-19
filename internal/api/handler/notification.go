package handler

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/service/notification"
)

type NotificationHandler struct {
	svc  *notification.NotificationService
	repo *repository.MetadataRepository
}

func NewNotificationHandler(svc *notification.NotificationService, repo *repository.MetadataRepository) *NotificationHandler {
	return &NotificationHandler{svc: svc, repo: repo}
}

func (h *NotificationHandler) SendWebhook(
	ctx context.Context,
	req *connect.Request[v1.SendWebhookRequest],
) (*connect.Response[v1.SendWebhookResponse], error) {
	err := h.svc.SendWebhook(req.Msg.Url, map[string]string{
		"payload": req.Msg.PayloadJson,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.SendWebhookResponse{
		Success: true,
	}), nil
}

func (h *NotificationHandler) ListChannels(
	ctx context.Context,
	req *connect.Request[v1.ListChannelsRequest],
) (*connect.Response[v1.ListChannelsResponse], error) {
	channels, err := h.repo.ListNotificationChannels(req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbChannels []*v1.NotificationChannel
	for _, c := range channels {
		pbChannels = append(pbChannels, &v1.NotificationChannel{
			Id:         c.ID,
			Name:       c.Name,
			Type:       c.Type,
			ConfigJson: c.ConfigJSON,
		})
	}

	return connect.NewResponse(&v1.ListChannelsResponse{
		Channels: pbChannels,
	}), nil
}
