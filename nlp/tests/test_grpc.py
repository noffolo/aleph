import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import nlp_pb2
import nlp_pb2_grpc
from main import NLPService, analyze_sentiment_simple
import grpc
from grpc_testing import server_from_dictionary, strict_fake_time


class FakeContext:
    def __init__(self):
        self._code = None
        self._details = None

    def set_code(self, code):
        self._code = code

    def set_details(self, details):
        self._details = details


class TestSentimentHeuristic:

    def test_positive(self):
        score, label = analyze_sentiment_simple("ottima crescita eccellente")
        assert label == "positive"
        assert score > 0

    def test_negative(self):
        score, label = analyze_sentiment_simple("pessimo fallimento crisi")
        assert label == "negative"
        assert score < 0

    def test_neutral(self):
        score, label = analyze_sentiment_simple("il tavolo e di legno")
        assert label == "neutral"
        assert -0.2 <= score <= 0.2

    def test_empty(self):
        score, label = analyze_sentiment_simple("")
        assert label == "neutral"
        assert score == 0.0

    def test_mixed(self):
        score, label = analyze_sentiment_simple("good growth but high risk")
        assert label in ("positive", "negative", "neutral")

    def test_returns_uncalibrated_scores(self):
        score, label = analyze_sentiment_simple("excellent great positive")
        assert -1.0 <= score <= 1.0
        assert label in ("positive", "negative", "neutral")

    def test_bilingual_italian(self):
        score, label = analyze_sentiment_simple("buono successo crescita")
        assert label == "positive"
        assert score > 0

    def test_bilingual_english(self):
        score, label = analyze_sentiment_simple("bad terrible decline")
        assert label == "negative"
        assert score < 0


class TestAnalyzeSentimentRPC:

    def test_positive_text(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="This is excellent and great!")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.label == "POSITIVE"
        assert resp.score > 0.5
        assert resp.method == "heuristic"
        assert resp.is_calibrated is False

    def test_negative_text(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="This is terrible and bad!")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.label == "NEGATIVE"
        assert resp.score < 0.5
        assert resp.method == "heuristic"
        assert resp.is_calibrated is False

    def test_neutral_text(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="The sky is blue.")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.label == "NEUTRAL"
        assert resp.method == "heuristic"
        assert resp.is_calibrated is False

    def test_empty_text(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.label == "NEUTRAL"
        assert resp.score == 0.5
        assert resp.method == "heuristic"
        assert resp.is_calibrated is False

    def test_method_field_always_heuristic(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="great success")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.method == "heuristic"
        assert resp.is_calibrated is False

    def test_is_calibrated_always_false(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="mixed good bad")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.is_calibrated is False

    def test_italian_text(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="Ottima crescita e successo!")
        ctx = FakeContext()
        resp = svc.AnalyzeEntiment(req, ctx) if hasattr(svc, 'AnalyzeEntiment') else None
        resp = svc.AnalyzeSentiment(req, ctx)
        assert resp.label == "POSITIVE"

    def test_response_has_new_fields(self):
        svc = NLPService.__new__(NLPService)
        svc.model = None
        svc.is_onnx = False
        svc._sentiment_method = "heuristic"
        req = nlp_pb2.AnalyzeSentimentRequest(text="excellent progress")
        ctx = FakeContext()
        resp = svc.AnalyzeSentiment(req, ctx)
        assert hasattr(resp, 'method')
        assert hasattr(resp, 'is_calibrated')
        assert isinstance(resp.score, float)
        assert isinstance(resp.label, str)
        assert 0.0 <= resp.score <= 1.0
        assert resp.label in ("POSITIVE", "NEGATIVE", "NEUTRAL")