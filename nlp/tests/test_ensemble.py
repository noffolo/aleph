"""Tests for NLP ensemble.py — PredictiveEnsemble, _data_hash."""
import sys
import os
import numpy as np
import pandas as pd
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from ensemble import _data_hash, PredictiveEnsemble


class TestDataHash:
    def test_same_df_same_hash(self):
        df1 = pd.DataFrame({"ds": pd.date_range("2024-01-01", periods=10, freq="D"),
                            "y": np.linspace(100, 110, 10)})
        df2 = pd.DataFrame({"ds": pd.date_range("2024-01-01", periods=10, freq="D"),
                            "y": np.linspace(100, 110, 10)})
        assert _data_hash(df1) == _data_hash(df2)

    def test_different_df_different_hash(self):
        df1 = pd.DataFrame({"ds": pd.date_range("2024-01-01", periods=10, freq="D"),
                            "y": np.linspace(100, 110, 10)})
        df2 = pd.DataFrame({"ds": pd.date_range("2024-01-01", periods=10, freq="D"),
                            "y": np.linspace(200, 220, 10)})
        assert _data_hash(df1) != _data_hash(df2)

    def test_hash_is_hex_string(self):
        df = pd.DataFrame({"ds": pd.date_range("2024-01-01", periods=5, freq="D"),
                           "y": [1.0, 2.0, 3.0, 4.0, 5.0]})
        h = _data_hash(df)
        assert isinstance(h, str)
        assert len(h) == 64  # SHA-256 hex digest
        assert all(c in "0123456789abcdef" for c in h)


class TestPredictProbs:
    def test_none_features_returns_point_five(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        assert ensemble.predict_probs(None) == 0.5

    def test_empty_dict_returns_point_five(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        assert ensemble.predict_probs({}) == 0.5

    def test_dict_with_string_values_returns_point_five(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        assert ensemble.predict_probs({"a": "string", "b": "also_string"}) == 0.5

    def test_dict_with_numeric_values(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        prob = ensemble.predict_probs({"a": 1.0, "b": 2.0})
        assert 0.01 <= prob <= 0.99
        assert isinstance(prob, float)

    def test_dict_mixed_skips_non_numeric(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        prob = ensemble.predict_probs({"a": 2.0, "b": "text", "c": True})
        assert 0.01 <= prob <= 0.99

    def test_empty_list_returns_point_five(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        assert ensemble.predict_probs([]) == 0.5

    def test_list_with_values(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        prob = ensemble.predict_probs([1.0, 2.0, 3.0])
        assert 0.01 <= prob <= 0.99

    def test_ndarray_with_values(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        prob = ensemble.predict_probs(np.array([1.0, 2.0, 3.0]))
        assert 0.01 <= prob <= 0.99

    def test_positive_values_give_high_prob(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        prob_high = ensemble.predict_probs([5.0, 10.0, 15.0])
        prob_low = ensemble.predict_probs([-5.0, -10.0, -15.0])
        assert prob_high > prob_low

    def test_clamped_to_range(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        # Very large positive → near 0.99
        prob_high = ensemble.predict_probs([100.0])
        assert prob_high <= 0.99
        # Very large negative → near 0.01
        prob_low = ensemble.predict_probs([-100.0])
        assert prob_low >= 0.01


class TestBrierScore:
    def test_perfect_predictions(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        score = ensemble.calculate_brier_score([1.0, 0.0, 1.0], [1.0, 0.0, 1.0])
        assert score == 0.0

    def test_worst_predictions(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        score = ensemble.calculate_brier_score([1.0, 0.0], [0.0, 1.0])
        assert score == 1.0

    def test_partial_accuracy(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        score = ensemble.calculate_brier_score([0.9, 0.1, 0.8], [1.0, 0.0, 1.0])
        assert 0.0 < score < 1.0

    def test_list_and_array_input(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        score_list = ensemble.calculate_brier_score([0.5, 0.5], [0.0, 1.0])
        score_array = ensemble.calculate_brier_score(
            np.array([0.5, 0.5]), np.array([0.0, 1.0]))
        assert score_list == score_array


class TestPredictiveEnsembleInit:
    def test_init_creates_cache(self):
        ensemble = PredictiveEnsemble()
        assert hasattr(ensemble, '_forecast_cache')
        assert ensemble._forecast_cache._name == "ensemble"

    def test_init_creates_simulator(self):
        ensemble = PredictiveEnsemble()
        assert hasattr(ensemble, 'simulator')


class TestForecastTrend:
    def test_short_history_returns_default(self):
        ensemble = PredictiveEnsemble.__new__(PredictiveEnsemble)
        df = pd.DataFrame({
            "ds": pd.date_range("2024-01-01", periods=5, freq="D"),
            "y": [1.0, 2.0, 3.0, 4.0, 5.0],
        })
        drift, sigma = ensemble.forecast_trend(df)
        assert drift == 0.05
        assert sigma == 0.2
