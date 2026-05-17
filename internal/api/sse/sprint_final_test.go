package sse

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type flusherWriter struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherWriter) Flush() {
	f.flushed = true
}

func (f *flusherWriter) Unwrap() http.ResponseWriter {
	return f.ResponseRecorder
}

func TestNewBroker_DefaultKeepalive(t *testing.T) {
	b := NewBroker(0, nil)
	assert.Equal(t, DefaultKeepaliveInterval, b.keepalive)
	b.Close()
}

func TestNewBroker_NilLogger(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	assert.NotNil(t, b.logger)
	b.Close()
}

func TestPublishTo_MissingClient(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	defer b.Close()

	assert.NotPanics(t, func() {
		b.PublishTo("nonexistent", "event_type", map[string]string{"msg": "hello"})
	})
}

func TestPublishTo_ExistingClient(t *testing.T) {
	b := NewBroker(time.Hour, nil)

	client := b.Subscribe("client-1")
	time.Sleep(10 * time.Millisecond)

	b.PublishTo("client-1", "direct_event", map[string]string{"targeted": "yes"})

	select {
	case evt := <-client.Events():
		assert.Equal(t, "direct_event", evt.Event)
		data, ok := evt.Data.(map[string]interface{})
		if ok {
			assert.Equal(t, "yes", data["targeted"])
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("did not receive event from PublishTo")
	}

	b.Unsubscribe("client-1")
}

func TestPublishTo_BufferFull(t *testing.T) {
	b := NewBroker(100*time.Millisecond, nil)
	defer b.Close()

	client := b.Subscribe("buf-client")
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < DefaultChannelBuffer+10; i++ {
		b.PublishTo("buf-client", "fill", i)
	}

	drained := 0
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case <-client.Events():
			drained++
		case <-timeout:
			break loop
		}
	}
	assert.Greater(t, drained, 0)
}

func TestBroker_Close_AlreadyClosed(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	b.Close()
	assert.NotPanics(t, func() { b.Close() })
}

func TestBroker_Publish_Broadcast(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	defer b.Close()

	c1 := b.Subscribe("c1")
	c2 := b.Subscribe("c2")
	time.Sleep(10 * time.Millisecond)

	b.Publish("broadcast_event", "broadcast_data")

	received := make(map[string]bool)
	timeout := time.After(500 * time.Millisecond)

	for i := 0; i < 2; i++ {
		select {
		case evt := <-c1.Events():
			if evt.Event == "broadcast_event" {
				received["c1"] = true
			}
		case evt := <-c2.Events():
			if evt.Event == "broadcast_event" {
				received["c2"] = true
			}
		case <-timeout:
			break
		}
	}

	assert.True(t, received["c1"], "client c1 should receive broadcast")
	assert.True(t, received["c2"], "client c2 should receive broadcast")
}

func TestBroker_Unsubscribe_RemovesClient(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	defer b.Close()

	b.Subscribe("to-remove")
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, b.ClientCount())

	b.Unsubscribe("to-remove")
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, b.ClientCount())
}

func TestBroker_Unsubscribe_Nonexistent(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	defer b.Close()

	assert.NotPanics(t, func() { b.Unsubscribe("ghost") })
}

func TestBroker_Subscribe_AfterClose(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	b.Close()
	c := b.Subscribe("after-close")
	assert.NotNil(t, c)
}

func TestStreamEvents_Basic(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	fw := &flusherWriter{ResponseRecorder: httptest.NewRecorder()}
	client := b.Subscribe("stream-test")
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)
	b.PublishTo("stream-test", "hello", map[string]string{"msg": "world"})

	err := StreamEvents(fw, req, client, slog.Default())
	assert.NoError(t, err)
	assert.True(t, fw.flushed, "response should have been flushed")
	assert.Equal(t, "text/event-stream", fw.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", fw.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", fw.Header().Get("Connection"))
}

func TestStreamEvents_ClientDone(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	fw := &flusherWriter{ResponseRecorder: httptest.NewRecorder()}
	client := b.Subscribe("done-test")
	time.Sleep(10 * time.Millisecond)

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)

	client.Close()

	err := StreamEvents(fw, req, client, slog.Default())
	assert.NoError(t, err)
}

func TestStreamEvents_ContextCancelled(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	fw := &flusherWriter{ResponseRecorder: httptest.NewRecorder()}
	client := b.Subscribe("ctx-test")
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)

	err := StreamEvents(fw, req, client, slog.Default())
	assert.NoError(t, err)
}

// nonFlushingWriter wraps an http.ResponseWriter but explicitly does NOT
// implement http.Flusher. Used to test the "streaming not supported" path.
type nonFlushingWriter struct {
	w http.ResponseWriter
}

func (n *nonFlushingWriter) Header() http.Header         { return n.w.Header() }
func (n *nonFlushingWriter) Write(b []byte) (int, error) { return n.w.Write(b) }
func (n *nonFlushingWriter) WriteHeader(code int)        { n.w.WriteHeader(code) }

func TestStreamEvents_NoFlusher(t *testing.T) {
	w := &nonFlushingWriter{w: httptest.NewRecorder()}
	client := &Client{ID: "no-flush", events: make(chan *Event, 10), done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // prevent infinite loop if flusher check fails
	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)

	err := StreamEvents(w, req, client, slog.Default())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming not supported")
}

func TestStreamEvents_EventWritten(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	fw := &flusherWriter{ResponseRecorder: httptest.NewRecorder()}
	client := b.Subscribe("event-test")
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)

	b.PublishTo("event-test", "test_event", map[string]string{"key": "value"})

	err := StreamEvents(fw, req, client, slog.Default())
	assert.NoError(t, err)
	body := fw.Body.String()
	assert.Contains(t, body, "event: test_event")
	assert.Contains(t, body, `"key":"value"`)
}

func TestStreamEvents_MultipleEvents(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	fw := &flusherWriter{ResponseRecorder: httptest.NewRecorder()}
	client := b.Subscribe("multi-test")
	time.Sleep(10 * time.Millisecond)

	b.PublishTo("multi-test", "event_a", "data-a")
	b.PublishTo("multi-test", "event_b", "data-b")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)

	err := StreamEvents(fw, req, client, slog.Default())
	assert.NoError(t, err)
	body := fw.Body.String()
	assert.Contains(t, body, "event: event_a")
	assert.Contains(t, body, "event: event_b")
}

func TestBroker_ConcurrentSubUnsub(t *testing.T) {
	b := NewBroker(time.Hour, nil)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cid := fmt.Sprintf("c%d", id)
			b.Subscribe(cid)
			time.Sleep(1 * time.Millisecond)
			b.Unsubscribe(cid)
		}(i)
	}
	wg.Wait()
}

func TestStreamEvents_RetryHeader(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	fw := &flusherWriter{ResponseRecorder: httptest.NewRecorder()}
	client := b.Subscribe("retry-test")
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)

	err := StreamEvents(fw, req, client, slog.Default())
	assert.NoError(t, err)
	body := fw.Body.String()
	assert.Contains(t, body, "retry: 3000")
}
