//go:build contract

package nlp_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
)

const sidecarDefaultAddr = "localhost:8001"

func sidecarAddr() string {
	if addr := os.Getenv("NLP_SIDECAR_ADDR"); addr != "" {
		return addr
	}
	return sidecarDefaultAddr
}

func sidecarReachable(t *testing.T) bool {
	addr := sidecarAddr()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Logf("NLP sidecar unreachable at %s: %v (skipping)", addr, err)
		return false
	}
	conn.Close()
	return true
}

func newNLPClient(t *testing.T) nlpconnect.NLPServiceClient {
	t.Helper()
	return nlpconnect.NewNLPServiceClient(
		http.DefaultClient,
		"http://"+sidecarAddr(),
		connect.WithGRPC(),
	)
}

func newHealthClient(t *testing.T) (grpc_health_v1.HealthClient, func()) {
	t.Helper()
	conn, err := grpc.NewClient(
		sidecarAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("gRPC dial failed: %v", err)
	}
	return grpc_health_v1.NewHealthClient(conn), func() { conn.Close() }
}

func TestNLPContract_HealthCheck(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client, cleanup := newHealthClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "aleph.nlp.v1.NLPService",
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("expected SERVING, got %v", resp.GetStatus())
	}

	respAll, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Check (all) failed: %v", err)
	}
	if respAll.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("all-services: expected SERVING, got %v", respAll.GetStatus())
	}
}

func TestNLPContract_AnalyzeSentiment(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
		Text: "I absolutely love this product, it is amazing and wonderful!",
	}))
	if err != nil {
		t.Fatalf("AnalyzeSentiment failed: %v", err)
	}

	msg := resp.Msg
	score := msg.GetScore()
	label := msg.GetLabel()
	method := msg.GetMethod()
	isCalibrated := msg.GetIsCalibrated()

	if score < -1.0 || score > 1.0 {
		t.Errorf("score %f out of range [-1,1]", score)
	}
	if label != "positive" && label != "negative" && label != "neutral" {
		t.Errorf("unexpected label %q", label)
	}
	if method != "heuristic" {
		t.Errorf("expected heuristic method, got %q", method)
	}
	if isCalibrated {
		t.Error("is_calibrated must be false for heuristic")
	}
	if label != "positive" {
		t.Errorf("positive text got label %q", label)
	}
	if score <= 0 {
		t.Errorf("positive text got score %f (expected > 0)", score)
	}
}

func TestNLPContract_AnalyzeSentiment_Negative(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
		Text: "This is terrible, horrible, awful. I hate it.",
	}))
	if err != nil {
		t.Fatalf("AnalyzeSentiment failed: %v", err)
	}

	if resp.Msg.GetLabel() != "negative" {
		t.Errorf("expected negative, got %q", resp.Msg.GetLabel())
	}
	if resp.Msg.GetScore() >= 0 {
		t.Errorf("expected negative score, got %f", resp.Msg.GetScore())
	}
}

func TestNLPContract_AnalyzeSentiment_EmptyText(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
		Text: "",
	}))
	if err != nil {
		t.Fatalf("AnalyzeSentiment (empty) failed: %v", err)
	}

	if resp.Msg.GetLabel() != "neutral" {
		t.Errorf("expected neutral, got %q", resp.Msg.GetLabel())
	}
	if resp.Msg.GetScore() != 0.0 {
		t.Errorf("expected score 0, got %f", resp.Msg.GetScore())
	}
}

func TestNLPContract_RecordFeedback(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.RecordFeedback(ctx, connect.NewRequest(&nlp.RecordFeedbackRequest{
		EntityId:        "test_entity_contract_001",
		IsCorrect:       true,
		CorrectionValue: "Test correction value",
		FeedbackType:    "prediction",
	}))
	if err != nil {
		t.Fatalf("RecordFeedback failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("expected success=true")
	}
}

func TestNLPContract_RecordFeedback_Negative(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.RecordFeedback(ctx, connect.NewRequest(&nlp.RecordFeedbackRequest{
		EntityId:        "test_entity_contract_002",
		IsCorrect:       false,
		CorrectionValue: "Expected different outcome",
		FeedbackType:    "action_proposal",
	}))
	if err != nil {
		t.Fatalf("RecordFeedback (negative) failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Error("expected success=true for incorrect feedback too")
	}
}

func TestNLPContract_StreamPredictions(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stream, err := client.StreamPredictions(ctx, connect.NewRequest(&nlp.StreamPredictionsRequest{
		ContextId:     "contract_test_context",
		OntologyQuery: "test_query",
	}))
	if err != nil {
		t.Fatalf("StreamPredictions failed: %v", err)
	}
	defer stream.Close()

	count := 0
	for stream.Receive() {
		count++
		msg := stream.Msg()

		if msg.GetEntityId() == "" {
			t.Errorf("pred %d: entity_id empty", count)
		}
		if p := msg.GetProbability(); p < 0.0 || p > 1.0 {
			t.Errorf("pred %d: probability %f out of [0,1]", count, p)
		}
		if msg.GetPredictedState() == "" {
			t.Errorf("pred %d: predicted_state empty", count)
		}
		if msg.GetExplanation() == "" {
			t.Errorf("pred %d: explanation empty", count)
		}

		if count >= 50 {
			break
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one prediction")
	}
	t.Logf("received %d predictions", count)
}

func TestNLPContract_CrossRPCCompatibility(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := client.RecordFeedback(ctx, connect.NewRequest(&nlp.RecordFeedbackRequest{
		EntityId:        "compat_test",
		IsCorrect:       true,
		CorrectionValue: "compat check",
		FeedbackType:    "prediction",
	}))
	if err != nil {
		t.Fatalf("RecordFeedback compat: %v", err)
	}

	stream, err := client.StreamPredictions(ctx, connect.NewRequest(&nlp.StreamPredictionsRequest{
		ContextId:     "compat_test",
		OntologyQuery: "compat",
	}))
	if err != nil {
		t.Fatalf("StreamPredictions compat: %v", err)
	}
	defer stream.Close()

	received := false
	if stream.Receive() {
		msg := stream.Msg()
		_ = msg.GetEntityId()
		_ = msg.GetProbability()
		_ = msg.GetPredictedState()
		_ = msg.GetExplanation()
		_ = msg.GetIsSynthetic()
		received = true
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if !received {
		t.Error("expected stream data after feedback")
	}
}

func TestNLPContract_ConnectionFailure(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
		Text: "test",
	}))
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestNLPContract_ResponseTypeIntegrity(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sResp, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
		Text: "integrity test",
	}))
	if err != nil {
		t.Fatalf("AnalyzeSentiment: %v", err)
	}
	var _ float32 = sResp.Msg.GetScore()
	var _ string = sResp.Msg.GetLabel()
	var _ string = sResp.Msg.GetMethod()
	var _ bool = sResp.Msg.GetIsCalibrated()

	fResp, err := client.RecordFeedback(ctx, connect.NewRequest(&nlp.RecordFeedbackRequest{
		EntityId: "i", IsCorrect: false, CorrectionValue: "", FeedbackType: "prediction",
	}))
	if err != nil {
		t.Fatalf("RecordFeedback: %v", err)
	}
	var _ bool = fResp.Msg.GetSuccess()

	stream, err := client.StreamPredictions(ctx, connect.NewRequest(&nlp.StreamPredictionsRequest{
		ContextId: "i_ctx", OntologyQuery: "",
	}))
	if err != nil {
		t.Fatalf("StreamPredictions: %v", err)
	}
	defer stream.Close()

	for stream.Receive() {
		var _ string = stream.Msg().GetEntityId()
		var _ float32 = stream.Msg().GetProbability()
		var _ string = stream.Msg().GetPredictedState()
		var _ string = stream.Msg().GetExplanation()
		var _ bool = stream.Msg().GetIsSynthetic()
		break
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	hClient, hCleanup := newHealthClient(t)
	defer hCleanup()
	hResp, err := hClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "aleph.nlp.v1.NLPService",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	var _ grpc_health_v1.HealthCheckResponse_ServingStatus = hResp.GetStatus()
}

func TestNLPContract_RequestWireFormat(t *testing.T) {
	if !sidecarReachable(t) {
		t.Skip("NLP sidecar not reachable")
	}

	client := newNLPClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("unicode", func(t *testing.T) {
		_, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
			Text: "Ciao mondo! 😊 ñ ü è — unicode payload.",
		}))
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
	})

	t.Run("long text", func(t *testing.T) {
		long := ""
		for i := 0; i < 100; i++ {
			long += "A long test sentence. "
		}
		_, err := client.AnalyzeSentiment(ctx, connect.NewRequest(&nlp.AnalyzeSentimentRequest{
			Text: long,
		}))
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
	})

	t.Run("feedback all fields", func(t *testing.T) {
		_, err := client.RecordFeedback(ctx, connect.NewRequest(&nlp.RecordFeedbackRequest{
			EntityId:        "wire_ent",
			IsCorrect:       false,
			CorrectionValue: "Dovrebbe essere negativo.",
			FeedbackType:    "action_proposal",
		}))
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
	})

	t.Run("stream ontology", func(t *testing.T) {
		stream, err := client.StreamPredictions(ctx, connect.NewRequest(&nlp.StreamPredictionsRequest{
			ContextId:     "wire_ctx",
			OntologyQuery: "risk_analysis:financial:market_volatility",
		}))
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		defer stream.Close()
		for stream.Receive() {
			_ = stream.Msg()
			break
		}
		if err := stream.Err(); err != nil {
			t.Fatalf("stream error: %v", err)
		}
	})
}
