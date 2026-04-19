import grpc
from concurrent import futures
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer
import numpy as np
import pandas as pd
import json
import os

import nlp_pb2
import nlp_pb2_grpc

from simulator import StochasticSimulator
from ensemble import PredictiveEnsemble
from markets import MarketPredictor

class NLPService(nlp_pb2_grpc.NLPServiceServicer):
    def __init__(self):
        # Caricamento modello ONNX (Ottimizzato per Wave 2)
        model_dir = "onnx_model"
        if not os.path.exists(model_dir):
            # Fallback se non ancora convertito
            print("[NLP] ONNX model not found, falling back to PyTorch...")
            self.tokenizer = AutoTokenizer.from_pretrained("sentence-transformers/all-MiniLM-L6-v2")
            from transformers import AutoModel
            self.model = AutoModel.from_pretrained("sentence-transformers/all-MiniLM-L6-v2")
            self.is_onnx = False
        else:
            print(f"[NLP] Loading ONNX model from {model_dir}...")
            self.tokenizer = AutoTokenizer.from_pretrained(model_dir)
            self.model = ORTModelForFeatureExtraction.from_pretrained(model_dir)
            self.is_onnx = True
        
        self.simulator = StochasticSimulator()
        self.ensemble = PredictiveEnsemble()
        self.markets = MarketPredictor()
        print("[NLP] Advanced Ensemble (Prophet/XGBoost) and Market Predictor loaded.")

    def AnalyzeSentiment(self, request, context):
        # Mock semplice per sentiment (esistente)
        score = 0.8 if "ottimo" in request.text.lower() else 0.2
        return nlp_pb2.AnalyzeSentimentResponse(score=score, label="POSITIVE" if score > 0.5 else "NEGATIVE")

    def RecordFeedback(self, request, context):
        """
        Riceve feedback dall'utente per ricalibrare i parametri di simulazione.
        """
        print(f"[NLP] Feedback received for {request.entity_id}: Correct={request.is_correct}")
        
        # In una fase reale, questo aggiornerebbe i pesi della GNN o i parametri mu/sigma
        # delle SDE (Stochastic Differential Equations).
        # Per ora, logghiamo l'azione di apprendimento.
        with open("nlp/feedback_log.jsonl", "a") as f:
            f.write(json.dumps({
                "entity_id": request.entity_id,
                "is_correct": request.is_correct,
                "user_correction": request.correction_value,
                "timestamp": time.time()
            }) + "\n")
        
        return nlp_pb2.FeedbackResponse(success=True)

    def StreamPredictions(self, request, context):
        """
        Invia predizioni e suggerisce azioni (Proposals) combinando Ensemble e Segnali di Mercato.
        """
        print(f"[NLP] Generating Scenario Proposals for: {request.context_id}")
        
        # 1. Recupero sentiment dei mercati (Polymarket/Metaculus)
        market_prob = self.markets.fetch_market_prob("polymarket", "example_market_id")
        
        # 2. Simulazione stocastica calibrata
        history_df = pd.DataFrame({
            'ds': pd.date_range(start='2026-01-01', periods=20, freq='D'),
            'y': np.linspace(100, 110, 20) + np.random.normal(0, 1, 20)
        })
        forecast = self.ensemble.generate_ensemble_forecast(100, history_df)
        stats = forecast["stats"]
        
        # Calibrazione finale usando il nuovo metodo generalizzato
        final_prob = self.markets.calibrate(forecast["drift_detected"], {"polymarket": market_prob})

        # Suggerimento Azione (XGBoost logic)
        if final_prob > 0.7:
            yield nlp_pb2.StreamPredictionsResponse(
                entity_id="ACTION_PROPOSAL_ENsemble",
                probability=final_prob,
                predicted_state="ACTION_REQUIRED",
                explanation=f"Rischio calibrato {final_prob:.2f} (Internal + Market). Azione suggerita per contrastare il drift rilevato."
            )

        # Risultati statistici della simulazione
        yield nlp_pb2.StreamPredictionsResponse(
            entity_id="PREDICTION_MEAN",
            probability=0.5,
            predicted_state="STABLE_TREND",
            explanation=f"Media attesa: {stats['p50']:.2f}. P90: {stats['p90']:.2f}"
        )

    def GenerateEmbedding(self, text):
        """Utility interna per trasformare testo in vettori (ONNX Optimized)"""
        inputs = self.tokenizer(text, return_tensors="pt", padding=True, truncation=True, max_length=512)
        if self.is_onnx:
            # ONNX Inference
            outputs = self.model(**inputs)
            # ORTModelForFeatureExtraction outputs are typically similar to HF models
            embeddings = outputs.last_hidden_state.mean(dim=1)
        else:
            # PyTorch Fallback
            with torch.no_grad():
                outputs = self.model(**inputs)
            embeddings = outputs.last_hidden_state.mean(dim=1)
            
        return embeddings.numpy().tolist()[0]

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    nlp_pb2_grpc.add_NLPServiceServicer_to_server(NLPService(), server)
    server.add_insecure_port('[::]:8001')
    print("[NLP] Predictive Service started on port 8001")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
