import numpy as np
import pandas as pd
from prophet import Prophet
from xgboost import XGBClassifier
from simulator import StochasticSimulator
import time

class PredictiveEnsemble:
    def __init__(self):
        self.simulator = StochasticSimulator()
        self.event_classifier = XGBClassifier()
        # In una fase reale, caricheremmo modelli pre-trained
        # Qui simuliamo l'integrazione
        print("[Ensemble] Prophet & XGBoost Ensemble initialized.")

    def forecast_trend(self, history_data):
        """
        Usa Prophet per estrarre il trend (drift) dai dati storici.
        history_data: DataFrame con colonne ['ds', 'y']
        """
        if len(history_data) < 10:
            return 0.05 # Default drift if not enough data
        
        m = Prophet(daily_seasonality=True)
        m.fit(history_data)
        
        # Prediciamo il prossimo periodo per calcolare il drift
        future = m.make_future_dataframe(periods=30)
        forecast = m.predict(future)
        
        # Calcoliamo il drift annualizzato dai risultati di Prophet
        y_start = forecast.iloc[-30]['yhat']
        y_end = forecast.iloc[-1]['yhat']
        drift = (y_end - y_start) / y_start
        return drift

    def predict_probs(self, features):
        """
        Usa XGBoost per classificare la probabilità di eventi critici.
        """
        # Mock: in un sistema reale usiamo features estratte da DuckDB
        # Qui restituiamo una probabilità basata sulla volatilità simulata
        return 0.85 

    def calculate_brier_score(self, predictions, actuals):
        """
        Calcola il Brier Score per valutare l'accuratezza delle predizioni probabilistiche.
        Brier Score = 1/N * sum((f_i - o_i)^2)
        Dove f_i è la probabilità predetta e o_i è l'esito reale (0 o 1).
        Punteggio più basso = migliore accuratezza.
        """
        predictions = np.array(predictions)
        actuals = np.array(actuals)
        return np.mean((predictions - actuals)**2)

    def generate_ensemble_forecast(self, S0, history_df, T_days=30):
        """
        Combina Prophet (Drift) e GBM (Volatilità).
        """
        mu = self.forecast_trend(history_df)
        sigma = 0.2 # Potrebbe essere stimato dai residui di Prophet
        
        paths = self.simulator.geometric_brownian_motion(S0, mu, sigma, T_days)
        stats = self.simulator.get_stats(paths)
        
        return {
            "paths": paths,
            "stats": stats,
            "drift_detected": mu,
            "model_source": "Prophet+GBM"
        }

if __name__ == "__main__":
    # Test rapido
    df = pd.DataFrame({
        'ds': pd.date_range(start='2026-01-01', periods=20, freq='D'),
        'y': np.linspace(100, 110, 20) + np.random.normal(0, 1, 20)
    })
    ensemble = PredictiveEnsemble()
    res = ensemble.generate_ensemble_forecast(110, df)
    print(f"[Ensemble Test] Drift: {res['drift_detected']:.4f}, P50: {res['stats']['p50']:.2f}")
