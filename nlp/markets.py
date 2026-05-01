import logging
import re
import os
import json
import requests
from abc import ABC, abstractmethod

logger = logging.getLogger(__name__)

# Environment-configurable API endpoints
_POLYMARKET_API_URL = os.environ.get("POLYMARKET_API_URL", "https://clob.polymarket.com/markets")
_METACULUS_API_URL = os.environ.get("METACULUS_API_URL", "https://www.metaculus.com/api2/questions")
_MARKET_REQUEST_TIMEOUT = int(os.environ.get("MARKET_REQUEST_TIMEOUT", "10"))


class MarketAPIError(Exception):
    """Raised when an external market API call fails."""
    pass


class MarketSource(ABC):
    @abstractmethod
    def fetch(self, identifier):
        pass


class PolymarketSource(MarketSource):
    API_URL = _POLYMARKET_API_URL

    def fetch(self, identifier):
        if not re.match(r'^[a-zA-Z0-9_-]{1,128}$', identifier):
            raise ValueError(f"Invalid identifier for Polymarket: {identifier}")
        try:
            resp = requests.get(f"{self.API_URL}/{identifier}", timeout=_MARKET_REQUEST_TIMEOUT)
            resp.raise_for_status()
            data = resp.json()
            if "outcomePrices" in data and data["outcomePrices"]:
                prices = data["outcomePrices"]
                if isinstance(prices, str):
                    prices = prices.split(",")
                return float(prices[0])
            if "probability" in data:
                return float(data["probability"])
            raise MarketAPIError(f"Polymarket response missing outcomePrices/probability for {identifier}")
        except (requests.RequestException, ValueError, json.JSONDecodeError) as e:
            raise MarketAPIError(f"Polymarket API error for {identifier}: {e}") from e


class MetaculusSource(MarketSource):
    API_URL = _METACULUS_API_URL

    def fetch(self, identifier):
        if not (isinstance(identifier, int) or (isinstance(identifier, str) and identifier.isdigit())):
            raise ValueError(f"Invalid identifier for Metaculus: {identifier}")
        try:
            resp = requests.get(f"{self.API_URL}/{identifier}/", timeout=_MARKET_REQUEST_TIMEOUT)
            resp.raise_for_status()
            data = resp.json()
            if "community_prediction" in data:
                cp = data["community_prediction"]
                if isinstance(cp, dict) and "full" in cp:
                    return float(cp["full"]["q1"])
                if isinstance(cp, (int, float)):
                    return float(cp)
            if "probability" in data:
                return float(data["probability"])
            raise MarketAPIError(f"Metaculus response missing probability for {identifier}")
        except (requests.RequestException, ValueError, json.JSONDecodeError) as e:
            raise MarketAPIError(f"Metaculus API error for {identifier}: {e}") from e


class MarketPredictor:
    def __init__(self):
        self.sources = {
            "polymarket": PolymarketSource(),
            "metaculus": MetaculusSource()
        }

    def register_source(self, name, source_instance: MarketSource):
        if not isinstance(source_instance, MarketSource):
            raise ValueError("Source must implement MarketSource interface")
        self.sources[name] = source_instance

    def fetch_market_prob(self, source_name, identifier):
        if source_name not in self.sources:
            raise ValueError(f"Source {source_name} not supported")
        result = self.sources[source_name].fetch(identifier)
        if result is not None:
            return result
        raise MarketAPIError(f"{source_name} returned no data for {identifier}")

    def calibrate(self, internal_prob, market_data):
        if not market_data:
            return internal_prob

        valid_probs = []
        for val in market_data.values():
            if val is not None:
                valid_probs.append(val)

        if not valid_probs:
            return internal_prob

        avg_market_prob = sum(valid_probs) / len(valid_probs)
        return (internal_prob * 0.4) + (avg_market_prob * 0.6)
