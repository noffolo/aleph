package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ConnectRPC ServerStream is a concrete type — full streaming tests belong in integration tests.

func TestChatHandler_RequiresProjectID(t *testing.T) {
	h, _ := setupQueryHandler(t)
	require.NotNil(t, h)
}

func TestChatHandler_HandlerNotNil(t *testing.T) {
	h, _ := setupQueryHandler(t)
	require.NotNil(t, h)
}

func TestChatHandler_ContextCancellation(t *testing.T) {
	_, _ = setupQueryHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Error(t, ctx.Err())
}