"""Tests for NLP cache.py — TTLCache with lazy eviction."""
import sys
import os
import time
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from cache import TTLCache


class FakeMonotonic:
    """Fake monotonic clock for deterministic TTL tests."""
    def __init__(self, start=0.0):
        self._t = start

    def __call__(self):
        return self._t

    def advance(self, seconds):
        self._t += seconds


class TestTTLCacheInit:
    def test_default_init(self):
        c = TTLCache()
        assert c.size == 0
        assert c._max_size == 50
        assert c._default_ttl == 300.0
        assert c._name == "cache"

    def test_custom_init(self):
        c = TTLCache(max_size=20, default_ttl=60.0, name="test-cache")
        assert c._max_size == 20
        assert c._default_ttl == 60.0
        assert c._name == "test-cache"

    def test_default_name_when_empty(self):
        c = TTLCache(max_size=10, default_ttl=30.0, name="")
        assert c._name == "cache"


class TestTTLCachePutGet:
    def test_put_and_get(self):
        c = TTLCache()
        c.put("a", 42)
        assert c.get("a") == 42

    def test_get_missing_key_returns_default(self):
        c = TTLCache()
        assert c.get("missing") is None
        assert c.get("missing", "fallback") == "fallback"

    def test_get_expired_key_returns_default(self, monkeypatch):
        fake_time = FakeMonotonic(0.0)
        monkeypatch.setattr(time, 'monotonic', fake_time)

        c = TTLCache(default_ttl=10.0)
        c.put("k", "v")
        assert c.get("k") == "v"

        # Advance past TTL
        fake_time.advance(11.0)
        assert c.get("k") is None
        assert c.get("k", "fallback") == "fallback"

    def test_get_not_yet_expired(self, monkeypatch):
        fake_time = FakeMonotonic(0.0)
        monkeypatch.setattr(time, 'monotonic', fake_time)

        c = TTLCache(default_ttl=10.0)
        c.put("k", "v")
        fake_time.advance(5.0)
        assert c.get("k") == "v"

    def test_put_overwrites_existing_key(self):
        c = TTLCache()
        c.put("a", 1)
        c.put("a", 2)
        assert c.get("a") == 2

    def test_put_with_per_key_ttl(self, monkeypatch):
        fake_time = FakeMonotonic(0.0)
        monkeypatch.setattr(time, 'monotonic', fake_time)

        c = TTLCache(default_ttl=100.0)
        c.put("short", "val", ttl=5.0)
        c.put("long", "val", ttl=50.0)

        fake_time.advance(10.0)
        assert c.get("short") is None
        assert c.get("long") == "val"

    def test_put_uses_default_ttl_when_none(self, monkeypatch):
        fake_time = FakeMonotonic(0.0)
        monkeypatch.setattr(time, 'monotonic', fake_time)

        c = TTLCache(default_ttl=20.0)
        c.put("k", "v", ttl=None)
        fake_time.advance(15.0)
        assert c.get("k") == "v"
        fake_time.advance(10.0)
        assert c.get("k") is None

    def test_get_moves_key_to_end_lru(self):
        c = TTLCache(max_size=2)
        c.put("a", 1)
        c.put("b", 2)
        c.get("a")  # access a, moves to end
        c.put("c", 3)  # evicts b (oldest)
        assert c.get("a") == 1
        assert c.get("b") is None
        assert c.get("c") == 3


class TestTTLCacheEviction:
    def test_evicts_oldest_when_full(self):
        c = TTLCache(max_size=2)
        c.put("first", 1)
        c.put("second", 2)
        c.put("third", 3)
        assert c.get("first") is None
        assert c.get("second") == 2
        assert c.get("third") == 3

    def test_eviction_respects_lru(self):
        c = TTLCache(max_size=2)
        c.put("a", 1)
        c.put("b", 2)
        c.get("a")  # makes 'a' fresh, 'b' oldest
        c.put("c", 3)
        assert c.get("a") == 1
        assert c.get("b") is None
        assert c.get("c") == 3


class TestTTLCacheRemove:
    def test_remove_existing_key(self):
        c = TTLCache()
        c.put("a", 1)
        c.put("b", 2)
        c.remove("a")
        assert c.get("a") is None
        assert c.get("b") == 2

    def test_remove_missing_key_no_error(self):
        c = TTLCache()
        c.remove("nonexistent")  # should not raise
        assert c.size == 0

    def test_remove_then_put_reuses_key(self):
        c = TTLCache()
        c.put("a", 1)
        c.remove("a")
        c.put("a", 99)
        assert c.get("a") == 99


class TestTTLCacheClear:
    def test_clear_removes_all_entries(self):
        c = TTLCache()
        c.put("a", 1)
        c.put("b", 2)
        c.clear()
        assert c.size == 0
        assert c.get("a") is None
        assert c.get("b") is None

    def test_clear_resets_stats(self):
        c = TTLCache()
        c.put("x", 1)
        c.get("x")  # hit
        c.get("y")  # miss
        c.clear()
        assert c.stats["hits"] == 0
        assert c.stats["misses"] == 0
        assert c.stats["hit_rate"] == 0.0


class TestTTLCacheSize:
    def test_size_starts_zero(self):
        c = TTLCache()
        assert c.size == 0

    def test_size_tracks_entries(self):
        c = TTLCache()
        assert c.size == 0
        c.put("a", 1)
        assert c.size == 1
        c.put("b", 2)
        assert c.size == 2
        c.remove("a")
        assert c.size == 1


class TestTTLCacheStats:
    def test_initial_stats(self):
        c = TTLCache(name="test")
        s = c.stats
        assert s["name"] == "test"
        assert s["hits"] == 0
        assert s["misses"] == 0
        assert s["hit_rate"] == 0.0
        assert s["size"] == 0
        assert s["max_size"] == 50

    def test_hit_rate_calculation(self):
        c = TTLCache()
        c.put("x", 1)
        c.get("x")  # hit
        c.get("x")  # hit
        c.get("y")  # miss
        s = c.stats
        assert s["hits"] == 2
        assert s["misses"] == 1
        assert s["hit_rate"] == round(2 / 3, 4)

    def test_hit_rate_zero_divisions(self):
        c = TTLCache()
        s = c.stats
        assert s["hit_rate"] == 0.0

    def test_stats_reflect_cleared_state(self):
        c = TTLCache(name="after-clear")
        c.put("a", 1)
        c.get("a")
        c.clear()
        s = c.stats
        assert s["hits"] == 0
        assert s["size"] == 0


class TestTTLCacheThreadSafety:
    def test_concurrent_put_and_get(self):
        import threading
        c = TTLCache(max_size=1000)
        errors = []

        def writer():
            try:
                for i in range(100):
                    c.put(f"k{i}", i)
            except Exception as e:
                errors.append(e)

        def reader():
            try:
                for _ in range(100):
                    c.get("k50", 0)
                    _ = c.size
                    _ = c.stats
            except Exception as e:
                errors.append(e)

        threads = [threading.Thread(target=writer) for _ in range(5)] + \
                  [threading.Thread(target=reader) for _ in range(5)]

        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert len(errors) == 0, f"Got errors: {errors}"
