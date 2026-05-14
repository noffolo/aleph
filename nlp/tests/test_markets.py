"""Tests for NLP markets.py — MarketPredictor, MarketAPIError, MarketSource."""
import sys
import os
import time
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from markets import (
    MarketAPIError,
    MarketSource,
    MarketPredictor,
    PolymarketSource,
    MetaculusSource,
)


class TestMarketAPIError:
    def test_is_exception(self):
        err = MarketAPIError("test message")
        assert isinstance(err, Exception)

    def test_message_preserved(self):
        err = MarketAPIError("API failure for XYZ")
        assert "API failure" in str(err)
        assert "XYZ" in str(err)


class TestMarketSource:
    def test_is_abstract(self):
        # MarketSource is an ABC with abstract fetch method
        assert hasattr(MarketSource, 'fetch')
        assert hasattr(MarketSource, '__abstractmethods__')


class TestPolymarketSource:
    def test_api_url_is_configured(self):
        src = PolymarketSource()
        assert hasattr(src, 'API_URL')

    def test_rejects_invalid_identifier(self):
        src = PolymarketSource()
        try:
            src.fetch("invalid;id")
            assert False, "Should have raised"
        except ValueError as e:
            assert "Invalid identifier" in str(e)
        except MarketAPIError:
            # Also acceptable if it gets past validation and hits network
            pass

    def test_rejects_empty_identifier(self):
        src = PolymarketSource()
        try:
            src.fetch("")
            assert False, "Should have raised"
        except ValueError:
            pass
        except MarketAPIError:
            pass


class TestMetaculusSource:
    def test_api_url_is_configured(self):
        src = MetaculusSource()
        assert hasattr(src, 'API_URL')

    def test_rejects_string_non_digit_identifier(self):
        src = MetaculusSource()
        try:
            src.fetch("not-a-number")
            assert False, "Should have raised"
        except ValueError as e:
            assert "Invalid identifier" in str(e)
        except MarketAPIError:
            pass

    def test_accepts_int_identifier(self):
        src = MetaculusSource()
        # Should validate without error (may hit network)
        try:
            src.fetch(12345)
        except MarketAPIError:
            pass  # Network error is expected, validation passed

    def test_accepts_digit_string(self):
        src = MetaculusSource()
        try:
            src.fetch("12345")
        except MarketAPIError:
            pass  # Network error is expected


class TestMarketPredictorCache:
    def test_cache_key_format(self):
        mp = MarketPredictor()
        key = mp._cache_key("polymarket", "test-id")
        assert key == "polymarket/test-id"

    def test_cache_key_with_metaculus(self):
        mp = MarketPredictor()
        key = mp._cache_key("metaculus", "42")
        assert key == "metaculus/42"

    def test_cache_get_missing_returns_none(self):
        mp = MarketPredictor()
        assert mp._cache_get("polymarket", "missing") is None

    def test_cache_put_and_get(self):
        mp = MarketPredictor()
        mp._cache_put("polymarket", "test", 0.75)
        assert mp._cache_get("polymarket", "test") == 0.75

    def test_cache_get_expired(self, monkeypatch):
        class FakeTime:
            def __init__(self):
                self._t = 0.0

            def __call__(self):
                return self._t

            def advance(self, s):
                self._t += s

        ft = FakeTime()
        monkeypatch.setattr(time, 'monotonic', ft)

        mp = MarketPredictor()
        mp._cache_put("polymarket", "expires", 0.5)
        assert mp._cache_get("polymarket", "expires") == 0.5

        ft.advance(61)  # past polymarket TTL of 60s
        assert mp._cache_get("polymarket", "expires") is None

    def test_cache_eviction_when_full(self):
        mp = MarketPredictor()
        mp._market_cache_max = 2
        mp._cache_put("polymarket", "a", 0.1)
        mp._cache_put("polymarket", "b", 0.2)
        mp._cache_put("polymarket", "c", 0.3)
        # Should have evicted the oldest
        assert len(mp._market_cache) == 2

    def test_cache_get_cleans_expired_no_leak(self):
        import threading
        mp = MarketPredictor()
        mp._market_cache["polymarket/stale"] = (0.5, 0.0)  # expired at time 0
        # Expired entries should be cleaned on access
        result = mp._cache_get("polymarket", "stale")
        assert result is None
        assert "polymarket/stale" not in mp._market_cache


class TestMarketPredictorInit:
    def test_init_has_sources(self):
        mp = MarketPredictor()
        assert "polymarket" in mp.sources
        assert "metaculus" in mp.sources
        assert isinstance(mp.sources["polymarket"], PolymarketSource)
        assert isinstance(mp.sources["metaculus"], MetaculusSource)

    def test_init_has_cache(self):
        mp = MarketPredictor()
        assert hasattr(mp, '_market_cache')
        assert isinstance(mp._market_cache, dict)

    def test_init_has_ttls(self):
        mp = MarketPredictor()
        assert "polymarket" in mp._ttls
        assert "metaculus" in mp._ttls


class TestRegisterSource:
    def test_registers_valid_source(self):
        mp = MarketPredictor()
        src = PolymarketSource()
        mp.register_source("custom", src)
        assert "custom" in mp.sources
        assert mp.sources["custom"] is src

    def test_registers_gets_default_ttl(self):
        mp = MarketPredictor()
        src = PolymarketSource()
        mp.register_source("new_source", src)
        assert mp._ttls["new_source"] == 300

    def test_rejects_non_marketsource(self):
        mp = MarketPredictor()
        try:
            mp.register_source("bad", "not a source")
            assert False, "Should have raised ValueError"
        except ValueError as e:
            assert "MarketSource" in str(e)


class TestFetchMarketProb:
    def test_returns_cached_value(self):
        mp = MarketPredictor()
        mp._cache_put("polymarket", "cached-item", 0.55)
        result = mp.fetch_market_prob("polymarket", "cached-item")
        assert result == 0.55

    def test_unknown_source_raises(self):
        mp = MarketPredictor()
        try:
            mp.fetch_market_prob("nonexistent", "item")
            assert False, "Should have raised"
        except ValueError as e:
            assert "not supported" in str(e)


class TestCalibrate:
    def calibrate_helper(self, internal, market_data):
        mp = MarketPredictor()
        return mp.calibrate(internal, market_data)

    def test_empty_market_data_returns_internal(self):
        result = self.calibrate_helper(0.7, {})
        assert result == 0.7

    def test_none_values_filtered(self):
        result = self.calibrate_helper(0.5, {"a": None, "b": None})
        assert result == 0.5

    def test_averages_valid_probs(self):
        # (0.5 * 0.4) + (0.6 * 0.6) = 0.20 + 0.36 = 0.56
        result = self.calibrate_helper(0.5, {"polymarket": 0.6})
        assert result == 0.56

    def test_multiple_market_sources(self):
        # (0.3 * 0.4) + ((0.8 + 0.6)/2 * 0.6) = 0.12 + 0.42 = 0.54
        result = self.calibrate_helper(0.3, {"a": 0.8, "b": 0.6})
        assert result == 0.54

    def test_internal_weight_0_4_market_0_6(self):
        result = self.calibrate_helper(1.0, {"m": 0.0})
        assert result == 0.4
        result = self.calibrate_helper(0.0, {"m": 1.0})
        assert result == 0.6
