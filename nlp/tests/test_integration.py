import sys
import os
import json
from unittest import mock
import numpy as np
import pandas as pd
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from ensemble import PredictiveEnsemble
from markets import (
    PolymarketSource, MetaculusSource,
    MarketPredictor, MarketAPIError,
)
from simulator import StochasticSimulator


class TestEnsembleIntegration:
    def test_forecast_trend_with_prophet(self):
        np.random.seed(42)
        ensemble = PredictiveEnsemble()
        df = pd.DataFrame({
            'ds': pd.date_range('2024-01-01', periods=30, freq='D'),
            'y': np.linspace(100, 130, 30) + np.random.normal(0, 2, 30),
        })
        drift, sigma = ensemble.forecast_trend(df)
        assert isinstance(drift, float)
        assert isinstance(sigma, float)
        assert sigma > 0

    def test_forecast_trend_is_cached(self):
        np.random.seed(42)
        ensemble = PredictiveEnsemble()
        df = pd.DataFrame({
            'ds': pd.date_range('2024-01-01', periods=15, freq='D'),
            'y': np.linspace(50, 60, 15) + np.random.normal(0, 1, 15),
        })
        drift1, sigma1 = ensemble.forecast_trend(df)
        drift2, sigma2 = ensemble.forecast_trend(df)
        assert drift1 == drift2
        assert sigma1 == sigma2

    def test_generate_ensemble_forecast(self):
        np.random.seed(42)
        ensemble = PredictiveEnsemble()
        df = pd.DataFrame({
            'ds': pd.date_range('2024-01-01', periods=20, freq='D'),
            'y': np.linspace(100, 120, 20) + np.random.normal(0, 1.5, 20),
        })
        result = ensemble.generate_ensemble_forecast(S0=120, history_df=df, T_days=10)
        assert 'paths' in result
        assert 'stats' in result
        assert 'drift_detected' in result
        assert 'sigma' in result
        assert 'model_source' in result
        assert result['model_source'] == 'Prophet+GBM'
        assert 'p50' in result['stats']
        assert result['paths'].ndim == 2


class TestPolymarketSourceMock:
    def test_parses_outcome_prices(self):
        src = PolymarketSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"outcomePrices": [0.62, 0.38]}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch("valid-id")
            assert result == 0.62

    def test_parses_string_prices(self):
        src = PolymarketSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"outcomePrices": "0.75,0.25"}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch("valid-id")
            assert result == 0.75

    def test_parses_probability_fallback(self):
        src = PolymarketSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"probability": 0.45}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch("valid-id")
            assert result == 0.45

    def test_raises_market_api_error_on_http_failure(self):
        src = PolymarketSource()
        import requests as req_mod
        mock_resp = mock.MagicMock()
        mock_resp.raise_for_status.side_effect = req_mod.HTTPError("500 Server Error")

        with mock.patch('requests.get', return_value=mock_resp):
            try:
                src.fetch("valid-id")
                assert False, "Should have raised"
            except MarketAPIError:
                pass

    def test_raises_on_empty_response(self):
        src = PolymarketSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            try:
                src.fetch("valid-id")
                assert False, "Should have raised"
            except MarketAPIError:
                pass


class TestMetaculusSourceMock:
    def test_parses_community_prediction_full(self):
        src = MetaculusSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {
            "community_prediction": {"full": {"q1": 0.55}}
        }
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch("42")
            assert result == 0.55

    def test_parses_community_prediction_scalar(self):
        src = MetaculusSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"community_prediction": 0.72}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch("42")
            assert result == 0.72

    def test_parses_probability_fallback(self):
        src = MetaculusSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"probability": 0.33}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch("42")
            assert result == 0.33

    def test_raises_on_empty_response(self):
        src = MetaculusSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            try:
                src.fetch("42")
                assert False, "Should have raised"
            except MarketAPIError:
                pass

    def test_accepts_int_identifier(self):
        src = MetaculusSource()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"probability": 0.5}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = src.fetch(12345)
            assert result == 0.5


class TestMarketPredictorFetchMock:
    def test_fetch_market_prob_uncached(self):
        mp = MarketPredictor()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"outcomePrices": [0.42, 0.58]}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result = mp.fetch_market_prob("polymarket", "some-id")
            assert result == 0.42

    def test_fetch_market_prob_cached_second_call(self):
        mp = MarketPredictor()
        mock_resp = mock.MagicMock()
        mock_resp.json.return_value = {"outcomePrices": [0.88, 0.12]}
        mock_resp.raise_for_status.return_value = None

        with mock.patch('requests.get', return_value=mock_resp):
            result1 = mp.fetch_market_prob("polymarket", "cached-id")
            result2 = mp.fetch_market_prob("polymarket", "cached-id")
        assert result1 == result2 == 0.88
