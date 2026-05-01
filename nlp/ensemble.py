import hashlib
import logging
from collections import OrderedDict

import numpy as np
import pandas as pd
from prophet import Prophet
from simulator import StochasticSimulator

logger = logging.getLogger(__name__)


def _data_hash(df):
    """Return SHA-256 hex digest of DataFrame values for cache key."""
    h = hashlib.sha256()
    h.update(df.to_records(index=False).tobytes())
    h.update(df.columns.tolist().__repr__().encode())
    return h.hexdigest()


class PredictiveEnsemble:
    def __init__(self):
        self.simulator = StochasticSimulator()
        self._forecast_cache = OrderedDict()
        self._max_cache_size = 10
        logger.info("Prophet Ensemble initialized.")

    def forecast_trend(self, history_data):
        if len(history_data) < 10:
            return 0.05, 0.2

        key = _data_hash(history_data)
        if key in self._forecast_cache:
            logger.info("Using cached Prophet model for data hash %s...", key[:12])
            return self._forecast_cache[key]

        logger.info("Fitting Prophet model for data hash %s...", key[:12])
        m = Prophet(daily_seasonality=True)
        m.fit(history_data)

        future = m.make_future_dataframe(periods=30)
        forecast = m.predict(future)

        y_start = forecast.iloc[-30]['yhat']
        y_end = forecast.iloc[-1]['yhat']
        drift = (y_end - y_start) / y_start

        residuals = history_data['y'].values - forecast.iloc[:len(history_data)]['yhat'].values
        sigma = float(np.std(residuals)) if len(residuals) > 0 else 0.2

        result = (drift, sigma)
        self._forecast_cache[key] = result
        logger.info("Cached Prophet result for data hash %s (drift=%.4f, sigma=%.4f)", key[:12], drift, sigma)

        if len(self._forecast_cache) > self._max_cache_size:
            self._forecast_cache.popitem(last=False)

        return result

    def predict_probs(self, features):
        if features is None:
            return 0.5
        if isinstance(features, dict):
            vals = [v for v in features.values() if isinstance(v, (int, float))]
            if vals:
                mean_val = sum(vals) / len(vals)
                prob = 1.0 / (1.0 + np.exp(-mean_val))
                return float(np.clip(prob, 0.01, 0.99))
            return 0.5
        if isinstance(features, (list, np.ndarray)):
            arr = np.array(features, dtype=float)
            if arr.size > 0:
                mean_val = float(np.mean(arr))
                prob = 1.0 / (1.0 + np.exp(-mean_val))
                return float(np.clip(prob, 0.01, 0.99))
            return 0.5
        return 0.5

    def calculate_brier_score(self, predictions, actuals):
        predictions = np.array(predictions)
        actuals = np.array(actuals)
        return np.mean((predictions - actuals) ** 2)

    def generate_ensemble_forecast(self, S0, history_df, T_days=30):
        mu, sigma = self.forecast_trend(history_df)

        paths = self.simulator.geometric_brownian_motion(S0, mu, sigma, T_days)
        stats = self.simulator.get_stats(paths)

        return {
            "paths": paths,
            "stats": stats,
            "drift_detected": mu,
            "sigma": sigma,
            "model_source": "Prophet+GBM"
        }

if __name__ == "__main__":
    df = pd.DataFrame({
        'ds': pd.date_range(start='2026-01-01', periods=20, freq='D'),
        'y': np.linspace(100, 110, 20) + np.random.normal(0, 1, 20)
    })
    ensemble = PredictiveEnsemble()
    res = ensemble.generate_ensemble_forecast(110, df)
    print(f"[Ensemble Test] Drift: {res['drift_detected']:.4f}, Sigma: {res['sigma']:.4f}, P50: {res['stats']['p50']:.2f}")
