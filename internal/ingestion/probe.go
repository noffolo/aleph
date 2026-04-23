package ingestion

import (
	"context"
	"fmt"
)

type ProbeResult struct {
	DataPath        string `json:"data_path"`
	PaginationType  string `json:"pagination_type"`
	NextPageField   string `json:"next_page_field"`
}

type ProbeRunner struct {
	LLMClient interface{} // Placeholder for LLM client (Ollama/OpenAI)
}

func (p *ProbeRunner) Probe(ctx context.Context, endpoint string) (*ProbeResult, error) {
	// 1. Fetch cold sample
	resp, err := safeHTTPClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 2. Call LLM to deduce mapping (MOCK for now)
	// In a real implementation, we would send 'body' to LLM with a strict prompt.
	return &ProbeResult{
		DataPath:       "data.items",
		PaginationType: "offset",
	}, nil
}

func (p *ProbeRunner) Execute(ctx context.Context, result *ProbeResult) error {
	// Execute the extraction blind and fast based on ProbeResult
	fmt.Printf("Executing extraction on path: %s\n", result.DataPath)
	return nil
}
