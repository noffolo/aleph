"""
Thread-safe TTL cache with lazy eviction and configurable max size.

No external dependencies — uses Python stdlib (time, threading, collections).

Usage:
    cache = TTLCache(max_size=100, default_ttl=300)
    cache.put("key", "value")
    val = cache.get("key")  # returns None if expired or missing
    val = cache.get("key", default="fallback")
"""

import time
import logging
import threading
from collections import OrderedDict

logger = logging.getLogger(__name__)


class TTLCache:
    """A thread-safe, TTL-aware cache with lazy eviction.

    Entries are evicted on access (get/put) when their TTL has expired.
    No proactive cleanup thread — purely lazy eviction.

    Thread-safe via threading.Lock to support concurrent gRPC workers.
    """

    def __init__(self, max_size: int = 50, default_ttl: float = 300.0, name: str = ""):
        self._max_size = max_size
        self._default_ttl = default_ttl
        self._name = name or "cache"
        self._lock = threading.Lock()
        self._data: OrderedDict = OrderedDict()
        self._hits = 0
        self._misses = 0

    def get(self, key: str, default=None):
        """Return cached value if key exists and TTL has not expired, else default."""
        with self._lock:
            if key not in self._data:
                self._misses += 1
                return default

            value, deadline = self._data[key]
            if time.monotonic() > deadline:
                del self._data[key]
                self._misses += 1
                if logger.isEnabledFor(logging.INFO):
                    logger.info(
                        "Cache '%s' MISS for %s (expired)",
                        self._name, key,
                    )
                return default

            self._data.move_to_end(key)
            self._hits += 1
            if logger.isEnabledFor(logging.INFO):
                logger.info(
                    "Cache '%s' HIT for %s",
                    self._name, key,
                )
            return value

    def put(self, key: str, value, ttl: float | None = None):
        """Insert or update a cache entry with optional per-entry TTL (default: _default_ttl)."""
        if ttl is None:
            ttl = self._default_ttl
        deadline = time.monotonic() + ttl
        with self._lock:
            if key not in self._data and len(self._data) >= self._max_size:
                self._data.popitem(last=False)
            self._data[key] = (value, deadline)
            self._data.move_to_end(key)

    def remove(self, key: str):
        """Remove a specific key from the cache, if present."""
        with self._lock:
            self._data.pop(key, None)

    def clear(self):
        """Remove all entries from the cache."""
        with self._lock:
            self._data.clear()
            self._hits = 0
            self._misses = 0

    @property
    def size(self) -> int:
        """Current number of entries in the cache."""
        with self._lock:
            return len(self._data)

    @property
    def stats(self) -> dict:
        """Return hit/miss statistics."""
        with self._lock:
            total = self._hits + self._misses
            hit_rate = self._hits / total if total > 0 else 0.0
            return {
                "name": self._name,
                "hits": self._hits,
                "misses": self._misses,
                "hit_rate": round(hit_rate, 4),
                "size": len(self._data),
                "max_size": self._max_size,
            }
