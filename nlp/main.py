import grpc
from concurrent import futures
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer
import numpy as np
import pandas as pd
import json
import os
import re
import time
import hashlib

import nlp_pb2
import nlp_pb2_grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from simulator import StochasticSimulator
from ensemble import PredictiveEnsemble
from markets import MarketPredictor

DUCKDB_PATH = os.environ.get("DUCKDB_PATH", "")

_SAFE_IDENTIFIER = re.compile(r'^[a-zA-Z0-9_-]+$')

def _check_identifier(name):
    if not _SAFE_IDENTIFIER.match(name):
        raise ValueError(f"invalid identifier: {name}")

def load_history_from_duckdb(context_id, duckdb_path):
    if not duckdb_path or not os.path.exists(duckdb_path):
        return None
    try:
        _check_identifier(context_id)
        import duckdb
        con = duckdb.connect(duckdb_path, read_only=True)
        table_name = context_id
        tables = con.execute("SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'").fetchall()
        table_names = [t[0] for t in tables]
        if table_name not in table_names:
            con.close()
            return None

        col_rows = con.execute(
            "SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? ORDER BY ordinal_position",
            [table_name]
        ).fetchall()

        date_col = None
        value_col = None
        for col_name, col_type in col_rows:
            cl = col_name.lower()
            ct = col_type.upper()
            if not date_col and ("DATE" in ct or "TIMESTAMP" in ct):
                date_col = col_name
            if not value_col and ("DOUBLE" in ct or "FLOAT" in ct or "DECIMAL" in ct or "INTEGER" in ct or "BIGINT" in ct):
                if cl != date_col:
                    value_col = col_name

        if not date_col or not value_col:
            con.close()
            return None

        _check_identifier(date_col)
        _check_identifier(value_col)
        query = f'SELECT "{date_col}" AS ds, "{value_col}" AS y FROM "{table_name}" ORDER BY "{date_col}" ASC LIMIT 500'
        rows = con.execute(query).fetchall()
        con.close()

        if len(rows) < 10:
            return None

        df = pd.DataFrame(rows, columns=["ds", "y"])
        if not pd.api.types.is_datetime64_any_dtype(df["ds"]):
            df["ds"] = pd.to_datetime(df["ds"], errors="coerce")
        df = df.dropna(subset=["ds", "y"])
        return df
    except Exception as e:
        print(f"[NLP] DuckDB read failed: {e}")
        return None


def generate_synthetic_history():
    return pd.DataFrame({
        'ds': pd.date_range(start='2026-01-01', periods=20, freq='D'),
        'y': np.linspace(100, 110, 20) + np.random.normal(0, 1, 20)
    })


class NLPService(nlp_pb2_grpc.NLPServiceServicer):
    def __init__(self):
        model_dir = "onnx_model"
        self.model = None
        self.is_onnx = False
        try:
            if not os.path.exists(model_dir):
                raise FileNotFoundError("ONNX model directory not found")
            print(f"[NLP] Loading ONNX model from {model_dir}...")
            self.tokenizer = AutoTokenizer.from_pretrained(model_dir)
            self.model = ORTModelForFeatureExtraction.from_pretrained(model_dir)
            self.is_onnx = True
        except Exception as e:
            print(f"[NLP] ONNX loading failed ({e}), falling back to PyTorch...")
            try:
                self.tokenizer = AutoTokenizer.from_pretrained("sentence-transformers/all-MiniLM-L6-v2")
                from transformers import AutoModel
                self.model = AutoModel.from_pretrained("sentence-transformers/all-MiniLM-L6-v2")
                self.is_onnx = False
            except Exception as e2:
                print(f"[NLP] PyTorch loading also failed ({e2}). Model unavailable.")
                self.model = None
                self.is_onnx = False

        self.simulator = StochasticSimulator()
        self.ensemble = PredictiveEnsemble()
        self.markets = MarketPredictor()
        self.duckdb_path = DUCKDB_PATH
        print("[NLP] Ensemble (Prophet/XGBoost) and Market Predictor loaded.")

    def AnalyzeSentiment(self, request, context):
        text = request.text
        if not text or not text.strip():
            return nlp_pb2.AnalyzeSentimentResponse(score=0.5, label="NEUTRAL")
        if self.model is not None:
            try:
                embedding = self.GenerateEmbedding(text)
                if embedding:
                    arr = np.array(embedding)
                    mean_val = float(np.mean(arr))
                    score = max(0.0, min(1.0, 0.5 + mean_val * 0.5))
                    label = "POSITIVE" if score > 0.6 else "NEGATIVE" if score < 0.4 else "NEUTRAL"
                    return nlp_pb2.AnalyzeSentimentResponse(score=score, label=label)
            except Exception as e:
                print(f"[NLP] Embedding-based sentiment failed ({e}), using heuristic")
        score = 0.8 if "ottimo" in text.lower() or "great" in text.lower() or "excellent" in text.lower() else 0.2 if "pessimo" in text.lower() or "terrible" in text.lower() else 0.5
        label = "POSITIVE" if score > 0.6 else "NEGATIVE" if score < 0.4 else "NEUTRAL"
        return nlp_pb2.AnalyzeSentimentResponse(score=score, label=label)

    def RecordFeedback(self, request, context):
        print(f"[NLP] Feedback received for {request.entity_id}: Correct={request.is_correct}")
        log_dir = os.path.dirname(os.path.abspath("feedback_log.jsonl"))
        os.makedirs(log_dir, exist_ok=True)
        try:
            with open("feedback_log.jsonl", "a") as f:
                f.write(json.dumps({
                    "entity_id": request.entity_id,
                    "is_correct": request.is_correct,
                    "user_correction": request.correction_value,
                    "timestamp": time.time()
                }) + "\n")
        except IOError as e:
            print(f"[NLP] Failed to write feedback log: {e}")
        return nlp_pb2.RecordFeedbackResponse(success=True)

    def StreamPredictions(self, request, context):
        context_id = request.context_id
        ontology_query = request.ontology_query
        print(f"[NLP] Generating Scenario Proposals for: {context_id}, query: {ontology_query}")
        try:
            market_prob = None
            if context_id:
                market_prob = self.markets.fetch_market_prob("polymarket", context_id)
            if market_prob is None:
                market_prob = 0.5

            history_df = load_history_from_duckdb(context_id, self.duckdb_path)
            data_source = "duckdb"
            if history_df is None:
                history_df = generate_synthetic_history()
                data_source = "synthetic"

            S0 = float(history_df["y"].iloc[-1])
            forecast = self.ensemble.generate_ensemble_forecast(S0, history_df)
            stats = forecast["stats"]

            features = {"drift": forecast["drift_detected"], "market_prob": market_prob}
            if ontology_query:
                features["query_signal"] = int(hashlib.sha256(ontology_query.encode()).hexdigest()[:8], 16) % 100 / 100.0

            event_prob = self.ensemble.predict_probs(features)
            final_prob = self.markets.calibrate(forecast["drift_detected"], {"polymarket": market_prob})

            is_synthetic = data_source == "synthetic"
            if final_prob > 0.7:
                yield nlp_pb2.StreamPredictionsResponse(
                    entity_id=f"ACTION_PROPOSAL_{context_id or 'ENSEMBLE'}",
                    probability=final_prob,
                    predicted_state="ACTION_REQUIRED",
                    explanation=f"Rischio calibrato {final_prob:.2f} (Internal + Market). Azione suggerita per contrastare il drift rilevato.",
                    is_synthetic=is_synthetic
                )

            yield nlp_pb2.StreamPredictionsResponse(
                entity_id=f"PREDICTION_{context_id or 'MEAN'}",
                probability=event_prob,
                predicted_state="STABLE_TREND",
                explanation=f"Media attesa: {stats['p50']:.2f}. P90: {stats['p90']:.2f}. Event prob: {event_prob:.2f}. Data source: {data_source}",
                is_synthetic=is_synthetic
            )
        except Exception as e:
            print(f"[NLP] StreamPredictions error: {e}")
            yield nlp_pb2.StreamPredictionsResponse(
                entity_id="ERROR",
                probability=0.0,
                predicted_state="ERROR",
                explanation=f"Prediction failed: {str(e)}",
                is_synthetic=False
            )

    def GenerateEmbedding(self, text):
        if self.model is None:
            return []
        tensor_fmt = "np" if self.is_onnx else "pt"
        inputs = self.tokenizer(text, return_tensors=tensor_fmt, padding=True, truncation=True, max_length=512)
        if self.is_onnx:
            outputs = self.model(**inputs)
            embeddings = outputs.last_hidden_state.mean(dim=1)
        else:
            import torch
            with torch.no_grad():
                outputs = self.model(**inputs)
            embeddings = outputs.last_hidden_state.mean(dim=1)

        return embeddings.numpy().tolist()[0]

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    nlp_service = NLPService()
    nlp_pb2_grpc.add_NLPServiceServicer_to_server(nlp_service, server)

    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set(
        "aleph.nlp.v1.NLPService",
        health_pb2.HealthCheckResponse.SERVING
    )

    address = os.environ.get("GRPC_SERVER_ADDRESS", "[::]:8001")
    server.add_insecure_port(address)
    print(f"[NLP] Predictive Service started on {address}")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
