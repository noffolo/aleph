import re
import os
import requests
from abc import ABC, abstractmethod

# Environment-configurable API endpoints
_POLYMARKET_API_URL = os.environ.get("POLYMARKET_API_URL", "https://clob.polymarket.com/markets")
_METACULUS_API_URL = os.environ.get("METACULUS_API_URL", "https://www.metaculus.com/api2/questions")
_MARKET_REQUEST_TIMEOUT = int(os.environ.get("MARKET_REQUEST_TIMEOUT", "10"))

class MarketSource(ABC):
    @abstractmethod
    def fetch(self, identifier):
        pass

class PolymarketSource(MarketSource):
    API_URL = _POLYMARKET_API_URL

    def fetch(self, identifier):
        if not re.match(r'^[a-zA-Z0-9_-]{1,128}$', identifier):
            print(f"[Markets] Invalid identifier for Polymarket: {identifier}")
            return None
        try:
            resp = requests.get(f"{self.API_URL}/{identifier}", timeout=_MARKET_REQUEST_TIMEOUT)
            resp.raise_for_status()
            data = resp.json()
            if "outcomePrices" in data and data["outcomePrices"]:
                prices = data["outcomePrices"]
                if isinstance(prices, str):
                    prices = prices.split(",")
                yes_price = float(prices[0])
                return yes_price
            if "probability" in data:
                return float(data["probability"])
        except Exception as e:
            print(f"[Markets] Polymarket API error for {identifier}: {e}")
        return None

class MetaculusSource(MarketSource):
    API_URL = _METACULUS_API_URL

    def fetch(self, identifier):
        if not (isinstance(identifier, int) or (isinstance(identifier, str) and identifier.isdigit())):
            print(f"[Markets] Invalid identifier for Metaculus: {identifier}")
            return None
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
        except Exception as e:
            print(f"[Markets] Metaculus API error for {identifier}: {e}")
        return None

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
        print(f"[Markets] {source_name} returned None for {identifier}, using neutral 0.5")
        return 0.5

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
