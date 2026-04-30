import grpc
import grpc_testing
import pytest
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import nlp_pb2
import nlp_pb2_grpc


class FakeNLPServicer(nlp_pb2_grpc.NLPServiceServicer):

    def AnalyzeSentiment(self, request, context):
        text = request.text
        if not text or not text.strip():
            return nlp_pb2.AnalyzeSentimentResponse(
                score=0.5, label="NEUTRAL",
                method="heuristic", is_calibrated=False
            )

        positive_keywords = ["good", "great", "excellent", "positive", "success", "growth", "ottimo", "buono"]
        negative_keywords = ["bad", "terrible", "negative", "failure", "crisis", "pessimo", "cattivo"]

        text_lower = text.lower()
        pos_count = sum(1 for kw in positive_keywords if kw in text_lower)
        neg_count = sum(1 for kw in negative_keywords if kw in text_lower)

        if pos_count > neg_count:
            return nlp_pb2.AnalyzeSentimentResponse(
                score=0.75, label="POSITIVE",
                method="heuristic", is_calibrated=False
            )
        elif neg_count > pos_count:
            return nlp_pb2.AnalyzeSentimentResponse(
                score=0.25, label="NEGATIVE",
                method="heuristic", is_calibrated=False
            )
        else:
            return nlp_pb2.AnalyzeSentimentResponse(
                score=0.5, label="NEUTRAL",
                method="heuristic", is_calibrated=False
            )

    def StreamPredictions(self, request, context):
        context_id = request.context_id or "test"
        yield nlp_pb2.StreamPredictionsResponse(
            entity_id=f"PREDICTION_{context_id}",
            probability=0.65,
            predicted_state="STABLE_TREND",
            explanation="Test prediction from fake server",
            is_synthetic=False
        )

    def RecordFeedback(self, request, context):
        return nlp_pb2.RecordFeedbackResponse(success=True)


@pytest.fixture
def grpc_stub():
    servicer = FakeNLPServicer()
    test_server = grpc_testing.server_from_dictionary(
        {nlp_pb2_grpc.NLPServiceServicer: servicer}
    )
    channel = grpc.insecure_channel('localhost:8001')
    stub = nlp_pb2_grpc.NLPServiceStub(channel)
    yield stub
    channel.close()