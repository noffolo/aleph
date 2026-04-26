package sse

import (
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SSE Broker tests are limited because Broker.run() goroutines never exit
// (run() has no quit channel). Each broker goroutine lives forever.
// We test the goroutine-free parts (WriteEvent, splitLines, Client)
// and keep broker tests to a minimum.

func TestNewBroker(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	assert.NotNil(t, b)
	assert.Equal(t, time.Hour, b.keepalive)
	b.Close()
}

func TestDefaults(t *testing.T) {
	assert.Equal(t, 30*time.Second, DefaultKeepaliveInterval)
	assert.Equal(t, 64, DefaultChannelBuffer)
	assert.Equal(t, 3000, DefaultRetryMS)
}

func TestWriteEvent(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{
		Event: "test_event",
		Data:  map[string]string{"hello": "world"},
		ID:    "evt-1",
		Retry: 3000,
	})
	require.NoError(t, err)
	body := w.Body.String()
	assert.Contains(t, body, "id: evt-1\n")
	assert.Contains(t, body, "event: test_event\n")
	assert.Contains(t, body, "retry: 3000\n")
	assert.Contains(t, body, `data: {"hello":"world"}`)
	assert.Contains(t, body, "\n\n")
}

func TestWriteKeepalive(t *testing.T) {
	w := httptest.NewRecorder()
	WriteKeepalive(w)
	assert.Equal(t, ": keepalive\n\n", w.Body.String())
}

func TestWriteEventNilData(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{Event: "no_data"})
	require.NoError(t, err)
	assert.NotContains(t, w.Body.String(), "data:")
}

func TestWriteEventKeepaliveEvent(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{Event: ":keepalive"})
	require.NoError(t, err)
	assert.Contains(t, w.Body.String(), ": keepalive")
}

func TestWriteEventRetry(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{Event: "x", Data: "data", Retry: 5000})
	require.NoError(t, err)
	assert.Contains(t, w.Body.String(), "retry: 5000\n")
}

func TestWriteEventMultiLine(t *testing.T) {
	w := httptest.NewRecorder()
	// The Data field for WriteEvent gets JSON-marshaled, so newlines become \\n
	err := WriteEvent(w, &Event{Event: "multi", Data: "line1\nline2\nline3"})
	require.NoError(t, err)
	body := w.Body.String()
	assert.Contains(t, body, "event: multi")
	// JSON encoding escapes newlines as \\n, so it's a single data line
	assert.Contains(t, body, "data: \"line1\\nline2\\nline3\"")
}

func TestWriteEventBadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{Data: make(chan int)})
	assert.Error(t, err)
}

func TestWriteEventDataOnly(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{Data: "plain string"})
	require.NoError(t, err)
	body := w.Body.String()
	assert.Contains(t, body, "data: \"plain string\"\n")
	assert.NotContains(t, body, "event:")
	assert.NotContains(t, body, "id:")
	assert.NotContains(t, body, "retry:")
}

func TestWriteEventIDOnly(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteEvent(w, &Event{ID: "evt-42"})
	require.NoError(t, err)
	assert.Contains(t, w.Body.String(), "id: evt-42\n")
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []string
	}{
		{"empty", "", []string{""}},
		{"single", "hello", []string{"hello"}},
		{"two", "a\nb", []string{"a", "b"}},
		{"trailing nl", "a\nb\n", []string{"a", "b"}},
		{"multi", "a\nb\nc\nd", []string{"a", "b", "c", "d"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, splitLines(tt.in))
		})
	}
}

func TestEventJSONRoundtrip(t *testing.T) {
	evt := &Event{Event: "t", Data: "raw", ID: "evt-42", Retry: 1000}
	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "t", decoded.Event)
	assert.Equal(t, "evt-42", decoded.ID)
	assert.Equal(t, 1000, decoded.Retry)
}

func TestWriteEventStructData(t *testing.T) {
	w := httptest.NewRecorder()
	type customData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	err := WriteEvent(w, &Event{Event: "custom", Data: customData{Name: "test", Value: 42}})
	require.NoError(t, err)
	assert.Contains(t, w.Body.String(), `data: {"name":"test","value":42}`)
}

func TestClientClose(t *testing.T) {
	c := &Client{ID: "test", events: make(chan *Event, 10), done: make(chan struct{})}

	select {
	case <-c.Done():
		t.Fatal("not done before Close")
	default:
	}

	c.Close()
	select {
	case <-c.Done():
	default:
		t.Fatal("should be done after Close")
	}
}

func TestClientCloseMultiple(t *testing.T) {
	c := &Client{ID: "test", events: make(chan *Event, 10), done: make(chan struct{})}
	c.Close()
	c.Close()
	c.Close()
}

func TestClientEvents(t *testing.T) {
	c := &Client{ID: "test", events: make(chan *Event, 10), done: make(chan struct{})}
	c.events <- &Event{Event: "hello"}
	evt := <-c.Events()
	assert.Equal(t, "hello", evt.Event)
}

func TestClientDone(t *testing.T) {
	c := &Client{ID: "test", events: make(chan *Event, 10), done: make(chan struct{})}
	c.Close()
	<-c.Done()
}

func TestBrokerGoroutineSafety(t *testing.T) {
	// Verify broker methods don't panic even after Close
	b := NewBroker(time.Hour, slog.Default())
	b.Subscribe("a")
	b.Close()
	assert.NotPanics(t, func() { b.Publish("x", nil) })
	assert.NotPanics(t, func() { b.PublishTo("a", "x", nil) })
	assert.NotPanics(t, func() { b.Unsubscribe("a") })
	assert.NotPanics(t, func() { b.Close() })
	assert.NotPanics(t, func() { b.Subscribe("b") })
}

func TestBrokerClientCount(t *testing.T) {
	// Close broker after test to clean up goroutine
	var cleanup []*Broker
	defer func() {
		for _, b := range cleanup {
			b.Close()
		}
	}()

	b := NewBroker(time.Hour, nil)
	cleanup = append(cleanup, b)

	assert.Equal(t, 0, b.ClientCount())
}

func TestBrokerClose(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	b.Subscribe("a")
	b.Close()

	// After Close, no further ops should panic
	b.Subscribe("b")
	b.Unsubscribe("c")
	b.Publish("e", nil)
	b.PublishTo("c", "e", nil)
}

func TestBrokerConcurrentPublish(t *testing.T) {
	b := NewBroker(time.Hour, nil)
	defer b.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Publish("evt", "data")
		}()
	}
	wg.Wait()
}
