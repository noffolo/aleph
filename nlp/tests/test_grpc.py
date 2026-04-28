import grpc
from grpc_testing import server
import pytest
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import nlp_pb2
import nlp_pb2_grpc
from conftest import FakeNLPServicer


class TestAnalyzeSentiment:
    """Tests for AnalyzeSentiment gRPC endpoint."""

    @pytest.fixture
    def test_server(self):
        """Create test server for each test."""
        servicer = FakeNLPServicer()
        test_server = server.from_dictionary(
            {nlp_pb2_grpc.NLPServiceServicer: servicer}
        )
        yield test_server

    def test_analyze_sentiment_positive(self, test_server):
        """Test positive sentiment detection."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="This is excellent and great!")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert result.label == "POSITIVE"
        assert result.score > 0.5

    def test_analyze_sentiment_negative(self, test_server):
        """Test negative sentiment detection."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="This is terrible and bad!")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert result.label == "NEGATIVE"
        assert result.score < 0.5

    def test_analyze_sentiment_neutral(self, test_server):
        """Test neutral sentiment detection."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="The sky is blue.")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert result.label == "NEUTRAL"

    def test_analyze_sentiment_empty(self, test_server):
        """Test handling of empty text."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert result.label == "NEUTRAL"
        assert result.score == 0.5

    def test_analyze_sentiment_mixed(self, test_server):
        """Test handling of mixed sentiment."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="Good growth but high risk")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert result.label in ("POSITIVE", "NEGATIVE", "NEUTRAL")

    def test_analyze_sentiment_italian(self, test_server):
        """Test Italian language sentiment detection."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="Ottima crescita e successo!")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert result.label == "POSITIVE"

    def test_grpc_endpoint_responds(self, test_server):
        """Test that the gRPC endpoint responds without error."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="test input")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert hasattr(result, 'score')
        assert hasattr(result, 'label')

    def test_analyze_sentiment_response_fields(self, test_server):
        """Test that response contains all expected fields."""
        request = nlp_pb2.AnalyzeSentimentRequest(text="excellent progress")
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.AnalyzeSentiment,
            (),
            request,
            None
        )
        result = response.result()
        assert hasattr(result, 'score')
        assert hasattr(result, 'label')
        assert isinstance(result.score, float)
        assert isinstance(result.label, str)
        assert 0.0 <= result.score <= 1.0
        assert result.label in ("POSITIVE", "NEGATIVE", "NEUTRAL")


class TestStreamPredictions:
    """Tests for StreamPredictions gRPC endpoint."""

    @pytest.fixture
    def test_server(self):
        """Create test server for each test."""
        servicer = FakeNLPServicer()
        test_server = server.from_dictionary(
            {nlp_pb2_grpc.NLPServiceServicer: servicer}
        )
        yield test_server

    def test_stream_predictions_yields_response(self, test_server):
        """Test that StreamPredictions yields at least one response."""
        request = nlp_pb2.StreamPredictionsRequest(context_id="test_context")
        response = test_server.invoke_unary_stream(
            nlp_pb2_grpc.NLPServiceServicer.StreamPredictions,
            (),
            request,
            None
        )
        results = list(response.result())
        assert len(results) >= 1
        first = results[0]
        assert hasattr(first, 'entity_id')
        assert hasattr(first, 'probability')
        assert hasattr(first, 'predicted_state')
        assert hasattr(first, 'is_synthetic')


class TestRecordFeedback:
    """Tests for RecordFeedback gRPC endpoint."""

    @pytest.fixture
    def test_server(self):
        """Create test server for each test."""
        servicer = FakeNLPServicer()
        test_server = server.from_dictionary(
            {nlp_pb2_grpc.NLPServiceServicer: servicer}
        )
        yield test_server

    def test_record_feedback_success(self, test_server):
        """Test that RecordFeedback returns success."""
        request = nlp_pb2.RecordFeedbackRequest(
            entity_id="test_entity",
            is_correct=True
        )
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.RecordFeedback,
            (),
            request,
            None
        )
        result = response.result()
        assert result.success is True

    def test_record_feedback_with_correction(self, test_server):
        """Test RecordFeedback with correction value."""
        request = nlp_pb2.RecordFeedbackRequest(
            entity_id="test_entity",
            is_correct=False,
            correction_value="corrected_value"
        )
        response = test_server.invoke_unary_unary(
            nlp_pb2_grpc.NLPServiceServicer.RecordFeedback,
            (),
            request,
            None
        )
        result = response.result()
        assert result.success is True