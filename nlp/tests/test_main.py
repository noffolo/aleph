"""Tests for NLP main.py — _check_identifier, load_history_from_duckdb,
RecordFeedback, AnalyzeSentimentRPC edge cases, NLPService init."""
import sys
import os
import json
import tempfile
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

import nlp_pb2
from main import (
    _check_identifier,
    analyze_sentiment_simple,
    load_history_from_duckdb,
    NLPService,
    _SAFE_IDENTIFIER,
)


class TestCheckIdentifier:
    def test_valid_identifiers(self):
        _check_identifier("abc123")
        _check_identifier("A_B-C")
        _check_identifier("test_table_42")
        _check_identifier("z")

    def test_invalid_identifiers(self):
        for bad in ["drop table", "test;id", "foo bar", "table-name;--",
                     "''", '"test"', "test!id", "test@id"]:
            try:
                _check_identifier(bad)
                assert False, f"Should have raised for: {bad}"
            except ValueError:
                pass

    def test_empty_string_raises(self):
        try:
            _check_identifier("")
            assert False, "Should have raised"
        except ValueError:
            pass


class TestSafeIdentifierPattern:
    def test_pattern_matches_alphanum(self):
        assert _SAFE_IDENTIFIER.match("abc123")
        assert _SAFE_IDENTIFIER.match("test_table")
        assert _SAFE_IDENTIFIER.match("A-B_C")

    def test_pattern_rejects_sql(self):
        assert not _SAFE_IDENTIFIER.match("test; DROP TABLE users")
        assert not _SAFE_IDENTIFIER.match("test' OR '1'='1")
        assert not _SAFE_IDENTIFIER.match("test/*comment*/")
        assert not _SAFE_IDENTIFIER.match("test table")

    def test_pattern_rejects_special_chars(self):
        assert not _SAFE_IDENTIFIER.match("test!")
        assert not _SAFE_IDENTIFIER.match("test@domain")
        assert not _SAFE_IDENTIFIER.match("test#tag")
        assert not _SAFE_IDENTIFIER.match("test$money")


class TestAnalyzeSentimentSimpleEdgeCases:
    def test_numeric_text(self):
        score, label = analyze_sentiment_simple("12345 67890")
        assert label == "neutral"

    def test_punctuation_text(self):
        score, label = analyze_sentiment_simple("!!! ??? ...")
        assert label == "neutral"
        assert score == 0.0

    def test_single_positive_word(self):
        score, label = analyze_sentiment_simple("excellent")
        assert label == "positive"
        assert score > 0

    def test_single_negative_word(self):
        score, label = analyze_sentiment_simple("terrible")
        assert label == "negative"
        assert score < 0

    def test_case_insensitive(self):
        score1, label1 = analyze_sentiment_simple("GOOD GREAT EXCELLENT")
        score2, label2 = analyze_sentiment_simple("good great excellent")
        assert score1 == score2


class TestLoadHistoryFromDuckDB:
    def test_nonexistent_path_returns_none(self):
        result = load_history_from_duckdb("test_context", "/nonexistent/path.duckdb")
        assert result is None

    def test_empty_path_returns_none(self):
        result = load_history_from_duckdb("test_context", "")
        assert result is None

    def test_with_invalid_identifier_raises(self):
        try:
            load_history_from_duckdb("invalid;id", "doesnt-matter")
        except ValueError:
            pass


class TestRecordFeedback:
    def _make_service(self):
        svc = NLPService.__new__(NLPService)
        svc.simulator = None
        svc.ensemble = None
        svc.markets = None
        return svc

    def test_successful_feedback(self):
        svc = self._make_service()
        req = nlp_pb2.RecordFeedbackRequest(
            entity_id="test_entity",
            is_correct=True,
            correction_value="0.8",
            feedback_type="manual",
        )

        class FakeContext:
            def set_code(self, code): pass
            def set_details(self, details): pass

        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.jsonl') as f:
            tmpfile = f.name

        import builtins
        real_open = builtins.open
        try:
            builtins.open = lambda *a, **kw: real_open(tmpfile, 'a') if True else None
            resp = svc.RecordFeedback(req, FakeContext())
            assert resp.success is True
        finally:
            builtins.open = real_open
            os.unlink(tmpfile)

    def test_failure_when_cant_write(self):
        svc = self._make_service()
        req = nlp_pb2.RecordFeedbackRequest(
            entity_id="test_entity",
            is_correct=False,
            correction_value="0.0",
            feedback_type="test",
        )

        class FakeContext:
            def __init__(self):
                self.code = None
                self.details = None
            def set_code(self, code):
                self.code = code
            def set_details(self, details):
                self.details = details

        import builtins
        old_open = builtins.open
        try:
            builtins.open = lambda *a, **kw: (_ for _ in ()).throw(OSError("disk full"))
            ctx = FakeContext()
            resp = svc.RecordFeedback(req, ctx)
            assert resp.success is False
        finally:
            builtins.open = old_open

