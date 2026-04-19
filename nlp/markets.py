import requests
from abc import ABC, abstractmethod

class MarketSource(ABC):
    @abstractmethod
    def fetch(self, identifier):
        pass

class PolymarketSource(MarketSource):
    def fetch(self, identifier):
        # Implementazione reale: request a API Polymarket
        return 0.68

class MetaculusSource(MarketSource):
    def fetch(self, identifier):
        # Implementazione reale: request a API Metaculus
        return 0.72

class MarketPredictor:
    def __init__(self):
        self.sources = {
            "polymarket": PolymarketSource(),
            "metaculus": MetaculusSource()
        }

    def register_source(self, name, source_instance: MarketSource):
        """Allows dynamic registration of new market sources."""
        if not isinstance(source_instance, MarketSource):
            raise ValueError("Source must implement MarketSource interface")
        self.sources[name] = source_instance

    def fetch_market_prob(self, source_name, identifier):
        if source_name not in self.sources:
            raise ValueError(f"Source {source_name} not supported")
        return self.sources[source_name].fetch(identifier)

    def calibrate(self, internal_prob, market_data):
        """
        Calibrazione Bayesiana generalizzata: combina internal_prob con 
        la media ponderata delle fonti di mercato attive.
        """
        if not market_data:
            return internal_prob

        market_probs = [val for val in market_data.values()]
        avg_market_prob = sum(market_probs) / len(market_probs)

        # Calibrazione: 40% modello, 60% mercato
        return (internal_prob * 0.4) + (avg_market_prob * 0.6)


