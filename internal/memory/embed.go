package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ErrEmptyInput is returned by ProcessText when the input text is empty.
var ErrEmptyInput = errors.New("empty input text")

const (
	chunkSize    = 512
	chunkOverlap = 128
	embedTimeout = 30 * time.Second
)

type ollamaEmbedReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResp struct {
	Embedding []float32 `json:"embedding"`
}

// Embedder generates embeddings via Ollama's embedding API.
type Embedder struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewEmbedder creates an Embedder that calls Ollama at the given base URL with the given model.
func NewEmbedder(baseURL, model string) *Embedder {
	if model == "" {
		model = "nomic-embed-text"
	}
	return &Embedder{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: embedTimeout},
	}
}

// Embed generates an embedding vector for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body := ollamaEmbedReq{Model: e.model, Prompt: text}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embeddings", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed API status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ollamaEmbedResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return result.Embedding, nil
}

// Chunk splits text into overlapping chunks of approximately chunkSize tokens.
func Chunk(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	start := 0
	for start < len(words) {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, chunk)

		if end == len(words) {
			break
		}
		start = end - chunkOverlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

// ProcessText splits text into chunks, embeds each, and returns chunk-embedding pairs.
func (e *Embedder) ProcessText(ctx context.Context, text string) ([]string, [][]float32, error) {
	chunks := Chunk(text)
	if len(chunks) == 0 {
		return nil, nil, ErrEmptyInput
	}

	embeddings := make([][]float32, 0, len(chunks))
	validChunks := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		emb, err := e.Embed(ctx, chunk)
		if err != nil {
			slog.Warn("embedding chunk failed, skipping", "error", err, "chunk_len", len(chunk))
			continue
		}
		embeddings = append(embeddings, emb)
		validChunks = append(validChunks, chunk)
	}

	return validChunks, embeddings, nil
}
