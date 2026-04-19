package middleware

import (
	"context"
	"fmt"
	"math"
	"connectrpc.com/connect"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
)

// PredictionValidatorMiddleware scarta dati malformati prima che raggiungano il client
func PredictionValidatorMiddleware(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}

		if p, ok := resp.Any().(*nlp.StreamPredictionsResponse); ok {
			if math.IsNaN(float64(p.Probability)) || p.Probability < 0 || p.Probability > 1 {
				return nil, connect.NewError(connect.CodeDataLoss, fmt.Errorf("invalid prediction probability: %f", p.Probability))
			}
		}
		return resp, nil
	}
}
