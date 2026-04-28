import grpc
from grpc_testing import server
import pytest
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import nlp_pb2
import nlp_pb2_grpc


class FakeNLPServicer(nlp_pb2_grpc.NLPServiceServicer):
    """Fake NLP service for testing without loading actual models."""

    def AnalyzeSentiment(self, request, context):
        """Return a default positive sentiment response."""
        text = request.text

        # Simple keyword-based sentiment for testing
        if not text or not text.strip():
            return nlp_pb2.AnalyzeSentimentResponse(score=0.5, label="NEUTRAL")

        positive_keywords = ["good", "great", "excellent", "positive", "success", "growth", "ottimo", "buono"]
        negative_keywords = ["bad", "terrible", "negative", "failure", "crisis", "pessimo", "cattivo"]

        text_lower = text.lower()
        pos_count = sum(1 for kw in positive_keywords if kw in text_lower)
        neg_count = sum(1 for kw in negative_keywords if kw in text_lower)

        if pos_count > neg_count:
            return nlp_pb2.AnalyzeSentimentResponse(score=0.75, label="POSITIVE")
        elif neg_count > pos_count:
            return nlp_pb2.AnalyzeSentimentResponse(score=0.25, label="NEGATIVE")
        else:
            return nlp_pb2.AnalyzeSentimentResponse(score=0.5, label="NEUTRAL")

    def StreamPredictions(self, request, context):
        """Yield a single prediction for testing."""
        context_id = request.context_id or "test"
        yield nlp_pb2.StreamPredictionsResponse(
            entity_id=f"PREDICTION_{context_id}",
            probability=0.65,
            predicted_state="STABLE_TREND",
            explanation="Test prediction from fake server",
            is_synthetic=False
        )

    def RecordFeedback(self, request, context):
        """Acknowledge feedback received."""
        return nlp_pb2.RecordFeedbackResponse(success=True)


@pytest.fixture
def grpc_stub():
    """Create a gRPC stub connected to a fake server for testing."""
    servicer = FakeNLPServicer()

    test_server = server.from_dictionary(
        {nlp_pb2_grpc.NLPServiceServicer: servicer}
    )
    channel = grpc.insecure_channel('localhost:8001')
    stub = nlp_pb2_grpc.NLPServiceStub(channel)
    yield stub
    channel.close()