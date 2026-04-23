package nlp_adapter

import (
	"context"

	"connectrpc.com/connect"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/ingestion"
)

// Adapter wraps *handler.NLPHandler to satisfy the ingestion.NLPAnalyzer interface.
type Adapter struct {
	NLPHandler *handler.NLPHandler
}

// AnalyzeSentiment implements the ingestion.NLPAnalyzer interface by delegating to the NLPHandler.
func (a *Adapter) AnalyzeSentiment(ctx context.Context, text string) (score float32, label string, err error) {
	req := connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{
		Text: text,
	})
	resp, err := a.NLPHandler.AnalyzeSentiment(ctx, req)
	if err != nil {
		return 0.0, "", err
	}
	return resp.Msg.Score, resp.Msg.Label, nil
}

var _ ingestion.NLPAnalyzer = (*Adapter)(nil)