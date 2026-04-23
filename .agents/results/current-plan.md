# Sentiment Reimagined — Execution Plan

## Problem Summary
1. MiniLM embeddings measure similarity, NOT sentiment — semantically wrong
2. Heuristic fallback: 4 hardcoded words ("ottimo"→0.8, "pessimo"→0.2)
3. Ingestion engine writes `0.0` for sentiment features — never calls NLP
4. Sentiment isolated: not tracked, not in predictions, not an agent tool
5. Label case mismatch: NLP returns "POSITIVE", frontend does `.toLowerCase()` — fragile

## Scope: Phase 1
Replace sentiment model, add batch API, wire ingestion, add agent tool, fix label bug.

---

## Task Board

### T1: Add BatchAnalyzeSentiment to proto
**Priority:** P0 | **Effort:** S | **Depends:** none | **Agent:** backend

**Changes:**
- `api/proto/aleph/nlp/v1/nlp.proto` — add:
  ```protobuf
  message SentimentResult {
    string text = 1;
    float score = 2;
    string label = 3;         // always lowercase: positive/negative/neutral/mixed
    float confidence = 4;     // model confidence 0-1
    string model_hint = 5;    // "general" | "financial" | "absa"
  }

  message BatchAnalyzeSentimentRequest {
    repeated string texts = 1;
    string model_hint = 2;    // "general" (default), "financial"
    int32 batch_size = 3;     // 0 = auto (64), max 128
  }

  message BatchAnalyzeSentimentResponse {
    repeated SentimentResult results = 1;
    string model_used = 2;
  }
  ```
  Also add `model_hint` to existing `AnalyzeSentimentRequest` (field 2).

- Regenerate:
  - `internal/api/proto/aleph/nlp/v1/nlp.pb.go`
  - `internal/api/proto/aleph/nlp/v1/nlpconnect/nlp.connect.go`
  - `nlp/nlp_pb2.py`, `nlp/nlp_pb2_grpc.py`
  - Frontend Connect types (buf generate)

**Acceptance:** `go build ./...` and `tsc --noEmit` pass with new proto types.

---

### T2: Implement SentimentRouter + multilingual model in NLP sidecar
**Priority:** P0 | **Effort:** L | **Depends:** T1 | **Agent:** backend

**Changes:**
- `nlp/convert_onnx.py` — change model from `sentence-transformers/all-MiniLM-L6-v2` to `tabularisai/multilingual-sentiment-analysis`. Export ONNX with `ORTModelForSequenceClassification`.
- `nlp/main.py`:
  - Remove `GenerateEmbedding` hack from `AnalyzeSentiment`
  - Add `SentimentRouter` class:
    ```python
    class SentimentRouter:
        def __init__(self):
            self.models = {}  # model_hint → (model, tokenizer)
        def analyze(self, texts, model_hint="general"):
            # loads on first call per hint
            # returns List[SentimentResult]
    ```
  - Label normalization: map model output labels to lowercase (`"POSITIVE"` → `"positive"`, `"Very Negative"` → `"negative"`, etc.)
  - 5-class → 3-class mapping: `very_positive+positive` → `positive`, `very_negative+negative` → `negative`, `neutral` → `neutral`
  - Score mapping: use softmax probability of positive class (0-1 scale)
  - Implement `BatchAnalyzeSentiment` RPC: chunk texts by `batch_size`, call `SentimentRouter.analyze()`, return results
  - Update `AnalyzeSentiment` to use `SentimentRouter` instead of embedding hack
  - Keep `GenerateEmbedding` method for backward compat but mark deprecated
- `nlp/requirements.txt` — add `onnxruntime>=1.17.0` (already there), verify `transformers>=4.38.0` covers `AutoModelForSequenceClassification`
- `nlp/Dockerfile` — update `convert_onnx.py` call to use new model

**Acceptance:** 
- `docker build` succeeds
- `AnalyzeSentiment("ottimo")` returns `label="positive"`, not heuristic
- `AnalyzeSentiment("terrible")` returns `label="negative"` with proper model confidence
- `BatchAnalyzeSentiment(["text1","text2"])` returns 2 results
- Labels always lowercase

---

### T3: Wire Go NLP handler + circuit breaker for batch
**Priority:** P0 | **Effort:** M | **Depends:** T1 | **Agent:** backend

**Changes:**
- `internal/api/handler/nlp.go` — add:
  ```go
  func (h *NLPHandler) BatchAnalyzeSentiment(
      ctx context.Context,
      req *connect.Request[nlp.BatchAnalyzeSentimentRequest],
  ) (*connect.Response[nlp.BatchAnalyzeSentimentResponse], error) {
  ```
- `internal/api/handler/breaker.go` — add `BatchAnalyzeSentiment` method to `CircuitBreakerClient` (same pattern as `AnalyzeSentiment`)
- `internal/api/handler/breaker.go` — update `NLPServiceClient` interface usage if needed
- Wire handler route in `main.go` (or wherever NLPServiceHandler is registered)

**Acceptance:** Go handler proxies batch request to sidecar, circuit breaker protects it.

---

### T4: Wire ingestion engine to call BatchAnalyzeSentiment
**Priority:** P0 | **Effort:** M | **Depends:** T3 | **Agent:** backend

**Changes:**
- `internal/ingestion/engine.go`:
  - Add `nlpClient` field to `Engine` struct (or use the `CircuitBreakerClient`)
  - `NewEngine` — accept NLP client as parameter (currently takes `nlpAddr string` but doesn't use it)
  - Replace `enrichPredictiveMetadata` line 276-279:
    ```go
    // Instead of: feature_type, feature_value = "sentiment_"+col, 0.0
    // Collect text columns, batch call BatchAnalyzeSentiment, store real scores
    ```
  - New flow:
    1. Identify text columns (len > 10, string type)
    2. Collect all text values from those columns
    3. Call `BatchAnalyzeSentiment` with chunked batches (64 per call)
    4. INSERT real `score`, `label`, `confidence` into `system_features`
    5. ALTER TABLE to add `_aleph_sentiment_score FLOAT`, `_aleph_sentiment_label VARCHAR` columns
  - Update `system_features` schema: add `feature_label VARCHAR` column
  - Handle sidecar unavailable gracefully (circuit breaker open → skip enrichment, log warning)

**Acceptance:** After ingestion, `system_features` contains real sentiment scores, not `0.0`.

---

### T5: Add analyze_sentiment agent tool
**Priority:** P1 | **Effort:** M | **Depends:** T3 | **Agent:** backend

**Changes:**
- `internal/api/handler/query.go` — add to `tools` slice in `Chat()` method:
  ```go
  {
      "type": "function",
      "function": map[string]interface{}{
          "name": "analyze_sentiment",
          "description": "Analyze sentiment of one or more texts. Returns sentiment label (positive/negative/neutral), confidence score, and per-text breakdown.",
          "parameters": map[string]interface{}{
              "type": "object",
              "properties": map[string]interface{}{
                  "texts": map[string]interface{}{
                      "type": "array",
                      "items": map[string]interface{}{"type": "string"},
                      "description": "Texts to analyze",
                  },
                  "model_hint": map[string]interface{}{
                      "type": "string",
                      "enum":         []string{"general", "financial"},
                      "default":      "general",
                  },
              },
              "required": []string{"texts"},
          },
      },
  },
  ```
- Add tool execution handler in the `for _, tc := range toolCalls` loop (after `search_data` handler, line ~517):
  ```go
  } else if tc.Function.Name == "analyze_sentiment" {
      texts, _ := tc.Function.Arguments["texts"].([]interface{})
      var textStrs []string
      for _, t := range texts { textStrs = append(textStrs, fmt.Sprintf("%v", t)) }
      modelHint := "general"
      if mh, ok := tc.Function.Arguments["model_hint"].(string); ok && mh != "" { modelHint = mh }
      res, err := h.nlpClient.BatchAnalyzeSentiment(ctx, connect.NewRequest(&nlp.BatchAnalyzeSentimentRequest{
          Texts: textStrs, ModelHint: modelHint,
      }))
      if err != nil { resultStr = "Error: " + err.Error() } else {
          jb, _ := json.Marshal(res.Msg.Results)
          resultStr = string(jb)
      }
  }
  ```
- Add `nlpClient` field to `QueryHandler` struct (or pass via constructor)
- Update `NewQueryHandler` to accept NLP client

**Acceptance:** Copilot agent can call `analyze_sentiment` tool and get real sentiment results.

---

### T6: Fix label case normalization at source
**Priority:** P0 | **Effort:** S | **Depends:** T2 | **Agent:** backend

**Changes:**
- `nlp/main.py` — ensure ALL sentiment labels returned are lowercase:
  ```python
  LABEL_MAP = {
      "POSITIVE": "positive", "Positive": "positive",
      "NEGATIVE": "negative", "Negative": "negative",
      "NEUTRAL": "neutral", "Neutral": "neutral",
      "MIXED": "mixed", "Mixed": "mixed",
      "Very Positive": "positive", "Very Negative": "negative",
  }
  def normalize_label(raw_label: str) -> str:
      return LABEL_MAP.get(raw_label, raw_label.lower())
  ```
- `frontend/src/components/OracleView.tsx` line 81 — remove `.toLowerCase()` (labels now guaranteed lowercase from source):
  ```typescript
  // Before: label: (res.label || 'neutral').toLowerCase()
  // After:  label: res.label || 'neutral'
  ```
  This makes the contract explicit: NLP always returns lowercase.

**Acceptance:** All sentiment labels lowercase from NLP source. Frontend no longer needs to `.toLowerCase()`.

---

### T7: Tests — unit + integration for sentiment pipeline
**Priority:** P1 | **Effort:** M | **Depends:** T2, T3, T4 | **Agent:** qa

**Test strategy:**

**Unit (Python):**
- `nlp/test_sentiment.py` — new file:
  - `test_single_text_returns_result`: AnalyzeSentiment for "ottimo" → positive, score > 0.5
  - `test_batch_texts_returns_results`: BatchAnalyzeSentiment for 5 texts → 5 results
  - `test_model_hint_selects_model`: `model_hint="financial"` loads different model path
  - `test_labels_always_lowercase`: assert all labels in `["positive","negative","neutral","mixed"]`
  - `test_empty_text_returns_neutral`: empty/whitespace → neutral, 0.5
  - `test_5class_to_3class_mapping`: "Very Positive" → "positive"
  - `test_batch_size_chunking`: 200 texts with batch_size=64 → multiple chunks

**Unit (Go):**
- `internal/api/handler/nlp_test.go` — new file:
  - `TestBatchAnalyzeSentiment_Proxy`: mock sidecar, verify proxy
  - `TestBatchAnalyzeSentiment_CircuitBreaker`: verify breaker opens after 3 failures

**Integration:**
- `internal/integration/e2e_test.go` — add:
  - `TestUsability_BatchSentimentAnalysis`: start sidecar, call batch endpoint, verify real labels
  - `TestUsability_SentimentAgentTool`: chat with agent, verify `analyze_sentiment` tool callable
  - `TestUsability_IngestionSentimentEnrichment`: ingest CSV, verify `system_features` has non-zero scores

**Frontend:**
- `frontend/src/components/OracleView.test.tsx` — add:
  - Test that sentiment result renders correct color for "positive"/"negative"/"neutral"

**Acceptance:** All new tests pass. `go test ./...` and `vitest run` green.

---

### T8: Update frontend OracleView for batch + confidence display
**Priority:** P2 | **Effort:** S | **Depends:** T6 | **Agent:** frontend

**Changes:**
- `frontend/src/components/OracleView.tsx`:
  - Show `confidence` field from sentiment result (if present)
  - Add sentiment history mini-chart (optional, low priority)
  - Add model selector dropdown: general vs financial (calls `analyzeSentiment` with `modelHint`)
- `frontend/src/api/proto/aleph/nlp/v1/nlp_connect.ts` — regenerated from proto (T1)

**Acceptance:** OracleView shows confidence score alongside sentiment label. Model selector works.

---

## Dependency Graph

```
T1 (proto) ──┬── T2 (NLP sidecar model) ── T6 (label fix) ── T8 (frontend)
             ├── T3 (Go handler) ──┬── T4 (ingestion wiring)
             │                    └── T5 (agent tool)
             └────────────────────────── T7 (tests)
```

**Critical path:** T1 → T2 → T6 (label fix unblocks frontend)

**Parallel tracks after T1:**
- Track A: T2 → T6 (NLP model + label fix)
- Track B: T3 → T4 + T5 (Go wiring, can start once T3 done)
- Track C: T7 (tests, can start once T2+T3 done)

---

## What to do NOW vs LATER

### NOW (Phase 1 — this plan)
| # | Task | Effort |
|---|------|--------|
| T1 | Proto: add BatchAnalyzeSentiment + model_hint | S |
| T2 | NLP sidecar: multilingual model + SentimentRouter | L |
| T3 | Go handler + circuit breaker for batch | M |
| T4 | Ingestion engine wiring (replace 0.0) | M |
| T5 | Agent tool: analyze_sentiment | M |
| T6 | Label normalization fix | S |
| T7 | Tests | M |
| T8 | Frontend confidence display | S |

**Total: ~4.5 dev-days**

### LATER (Phase 2 — not in this plan)
- Financial sentiment model: `tabularisai/ModernFinBERT`
- ABSA: `yangheng/deberta-v3-base-absa-v1.1`
- Sentiment as prediction signal in `StreamPredictions`
- Sentiment drift detection over time
- Sentiment history chart in OracleView
- Prediction calibration using sentiment features

---

## Proto API Additions Summary

```protobuf
// New messages
message SentimentResult {
  string text = 1;
  float score = 2;
  string label = 3;         // always lowercase
  float confidence = 4;
  string model_hint = 5;
}

message BatchAnalyzeSentimentRequest {
  repeated string texts = 1;
  string model_hint = 2;
  int32 batch_size = 3;
}

message BatchAnalyzeSentimentResponse {
  repeated SentimentResult results = 1;
  string model_used = 2;
}

// New RPC added to NLPService
rpc BatchAnalyzeSentiment(BatchAnalyzeSentimentRequest) returns (BatchAnalyzeSentimentResponse);

// Existing message updated
message AnalyzeSentimentRequest {
  string text = 1;
  string model_hint = 2;  // NEW
}
```

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| Multilingual model too slow for batch | Medium | ONNX runtime, batch_size=64, benchmark first |
| ONNX export fails for new model | Low | Test export in T2, fallback to PyTorch |
| Proto regen breaks existing code | Medium | Additive-only changes, no field renames |
| Sidecar unavailable during ingestion | Medium | Circuit breaker → skip enrichment, log warning |
| 5-class → 3-class mapping loses info | Low | Store raw 5-class label in `feature_label` column |
