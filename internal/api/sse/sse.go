// Package sse provides Server-Sent Events utilities for real-time server→client push.
//
// SSE is chosen over gRPC server-streaming for notification/status use cases because:
//   - Unidirectional: only server→client, no client→server streaming needed
//   - Simpler client: native EventSource browser API, no gRPC-Web dependency
//   - HTTP infrastructure: works through standard HTTP proxies/load balancers/CDNs
//   - Auto-reconnect: EventSource has built-in Last-Event-ID reconnection
//   - No extra dependency: browsers have native EventSource support
//
// gRPC streaming (connect-go) is retained for:
//   - Chat (bidirectional token streaming + tool calls)
//   - StreamPredictions (server-streaming proxied from Python sidecar)
//
// These require bidirectional or protocol-level semantics that SSE cannot provide.
package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultKeepaliveInterval is how often the server sends a keepalive comment
	// to prevent proxies from closing idle connections.
	DefaultKeepaliveInterval = 30 * time.Second

	// DefaultChannelBuffer is the buffer size per client event channel.
	DefaultChannelBuffer = 64

	// DefaultRetryMS is the default reconnection time sent to the client in ms.
	DefaultRetryMS = 3000
)

// Event is a structured SSE event. Fields map directly to the SSE spec fields.
type Event struct {
	// Event is the event type (maps to "event:" line).
	// The client listens for this type via addEventListener(type, ...).
	Event string `json:"event,omitempty"`

	// Data is the JSON payload (maps to "data:" line(s)).
	Data interface{} `json:"data"`

	// ID sets the Last-Event-ID for reconnection resilience (maps to "id:" line).
	ID string `json:"id,omitempty"`

	// Retry advises the client on reconnection delay in milliseconds (maps to "retry:" line).
	Retry int `json:"retry,omitempty"`
}

// Client represents a single SSE consumer connection.
type Client struct {
	ID     string
	events chan *Event
	done   chan struct{}
	once   sync.Once
}

// Broker manages multiple SSE client connections and provides
// a thread-safe broadcast mechanism for pushing events to all connected clients.
type Broker struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan string
	broadcast  chan *Event
	keepalive  time.Duration
	logger     *slog.Logger
	mu         sync.RWMutex
	closed     atomic.Bool
	eventID    atomic.Int64
	quit       chan struct{}
}

// NewBroker creates a new SSE broker with the given keepalive interval.
// Pass 0 to use the default interval (30s).
func NewBroker(keepalive time.Duration, logger *slog.Logger) *Broker {
	if keepalive <= 0 {
		keepalive = DefaultKeepaliveInterval
	}
	if logger == nil {
		logger = slog.Default()
	}
	b := &Broker{
		clients:   make(map[string]*Client),
		register:  make(chan *Client, 16),
		unregister: make(chan string, 16),
		broadcast: make(chan *Event, 256),
		keepalive: keepalive,
		logger:    logger,
		quit:      make(chan struct{}),
	}
	go b.run()
	return b
}

// run is the central event loop: handles registration, unregistration,
// broadcasting, and keepalive pings.
func (b *Broker) run() {
	keepaliveTicker := time.NewTicker(b.keepalive)
	defer keepaliveTicker.Stop()

	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client.ID] = client
			b.mu.Unlock()
			b.logger.Debug("sse client connected", "id", client.ID, "total", len(b.clients))

		case id := <-b.unregister:
			b.mu.Lock()
			if c, ok := b.clients[id]; ok {
				c.Close()
				delete(b.clients, id)
			}
			b.mu.Unlock()
			b.logger.Debug("sse client disconnected", "id", id, "total", len(b.clients))

		case event := <-b.broadcast:
			b.mu.RLock()
			for _, client := range b.clients {
				select {
				case client.events <- event:
				default:
					// Client buffer full; skip to avoid blocking broker.
					b.logger.Warn("sse client buffer full, dropping event", "id", client.ID)
				}
			}
			b.mu.RUnlock()

		case <-keepaliveTicker.C:
			b.mu.RLock()
			for _, client := range b.clients {
				select {
				case client.events <- &Event{Event: ":keepalive"}:
				default:
				}
			}
			b.mu.RUnlock()

		case <-b.quit:
			b.logger.Info("sse broker run loop exiting")
			return
		}
	}
}

// Subscribe creates and registers a new SSE client. The caller must handle
// the returned channel lifecycle (done is embedded). Returns a Client that
// is already registered.
func (b *Broker) Subscribe(id string) *Client {
	c := &Client{
		ID:     id,
		events: make(chan *Event, DefaultChannelBuffer),
		done:   make(chan struct{}),
	}
	b.register <- c
	return c
}

// Unsubscribe removes a client by ID and cleans up its channel.
func (b *Broker) Unsubscribe(id string) {
	b.unregister <- id
}

// Publish sends an event to all connected clients asynchronously.
// It assigns a monotonically increasing ID for Last-Event-ID support.
func (b *Broker) Publish(eventType string, data interface{}) {
	eventID := fmt.Sprintf("evt-%d", b.eventID.Add(1))
	b.broadcast <- &Event{
		Event: eventType,
		Data:  data,
		ID:    eventID,
	}
}

// PublishTo sends an event to a specific client by ID.
func (b *Broker) PublishTo(clientID string, eventType string, data interface{}) {
	eventID := fmt.Sprintf("evt-%d", b.eventID.Add(1))
	b.mu.RLock()
	defer b.mu.RUnlock()
	if c, ok := b.clients[clientID]; ok {
		select {
		case c.events <- &Event{Event: eventType, Data: data, ID: eventID}:
		default:
			b.logger.Warn("sse client buffer full, dropping targeted event", "id", clientID)
		}
	}
}

// ClientCount returns the current number of connected SSE clients.
func (b *Broker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Close shuts down the broker and all client connections.
func (b *Broker) Close() {
	b.closed.Store(true)
	select {
	case <-b.quit:
	default:
		close(b.quit)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, c := range b.clients {
		c.Close()
		delete(b.clients, id)
	}
}

// Close signals the client's event loop to stop. Safe to call multiple times.
func (c *Client) Close() {
	c.once.Do(func() {
		close(c.done)
	})
}

// Done returns a channel that is closed when the client disconnects.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Events returns the event channel for the client.
func (c *Client) Events() <-chan *Event {
	return c.events
}

// WriteEvent writes a single SSE event to the http.ResponseWriter and flushes.
// It formats the event according to the SSE spec:
//
//	id: <id>\n
//	event: <event>\n
//	data: <json-data>\n
//	retry: <retry>\n
//	\n
func WriteEvent(w http.ResponseWriter, evt *Event) error {
	if evt.ID != "" {
		fmt.Fprintf(w, "id: %s\n", evt.ID)
	}
	if evt.Event != "" && evt.Event != ":keepalive" {
		fmt.Fprintf(w, "event: %s\n", evt.Event)
	}
	if evt.Retry > 0 {
		fmt.Fprintf(w, "retry: %d\n", evt.Retry)
	}
	if evt.Data != nil {
		data, err := json.Marshal(evt.Data)
		if err != nil {
			return fmt.Errorf("sse marshal event data: %w", err)
		}
		// SSE spec: split multi-line data into multiple "data:" lines
		lines := splitLines(string(data))
		for _, line := range lines {
			fmt.Fprintf(w, "data: %s\n", line)
		}
	} else if evt.Event == ":keepalive" {
		// Keepalive comment line — clients ignore these
		fmt.Fprintf(w, ": keepalive\n")
	}
	fmt.Fprintf(w, "\n")
	return nil
}

// WriteKeepalive writes an SSE keepalive comment and flushes.
func WriteKeepalive(w http.ResponseWriter) {
	fmt.Fprintf(w, ": keepalive\n\n")
}

// splitLines splits a string into lines for SSE multi-line data encoding.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	if len(lines) == 0 {
		lines = append(lines, s)
	}
	return lines
}

// StreamEvents reads from a client's event channel and writes SSE-formatted
// events to the http.ResponseWriter until the client disconnects or the
// context is cancelled. It sets up the response headers for SSE.
func StreamEvents(w http.ResponseWriter, r *http.Request, client *Client, logger *slog.Logger) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported: response writer does not implement http.Flusher")
	}

	// SSE required headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Send initial retry directive
	fmt.Fprintf(w, "retry: %d\n\n", DefaultRetryMS)
	flusher.Flush()

	logger.Debug("sse stream started", "client_id", client.ID)

	for {
		select {
		case <-r.Context().Done():
			logger.Debug("sse client context cancelled", "client_id", client.ID)
			return nil

		case <-client.Done():
			logger.Debug("sse client done signal received", "client_id", client.ID)
			return nil

		case evt, ok := <-client.Events():
			if !ok {
				return nil
			}
			if err := WriteEvent(w, evt); err != nil {
				logger.Error("sse write event failed", "client_id", client.ID, "error", err)
				return fmt.Errorf("streamEvents: %w", err)
			}
			flusher.Flush()
		}
	}
}
