import numpy as np

class StochasticSimulator:
    def __init__(self, dt=1/252): # Passo temporale giornaliero (anno finanziario)
        self.dt = dt

    def geometric_brownian_motion(self, S0, mu, sigma, T_days, num_sims=100):
        """
        Simula N percorsi futuri usando il Moto Browniano Geometrico.
        S0: Valore iniziale
        mu: Drift (tendenza)
        sigma: Volatilità
        T_days: Orizzonte temporale in giorni
        """
        n_steps = int(T_days)
        # Generazione rumore gaussiano per tutte le simulazioni in una volta (vettorializzato)
        dW = np.random.normal(0, np.sqrt(self.dt), (num_sims, n_steps))
        
        # Calcolo dei rendimenti logaritmici
        # dS = S * (mu * dt + sigma * dW)
        returns = (mu - 0.5 * sigma**2) * self.dt + sigma * dW
        cumulative_returns = np.cumsum(returns, axis=1)
        
        # Prezzi simulati
        paths = S0 * np.exp(cumulative_returns)
        # Inseriamo il punto di partenza
        paths = np.insert(paths, 0, S0, axis=1)
        
        return paths

    def get_stats(self, paths):
        """Restituisce intervalli di confidenza (P10, P50, P90)"""
        last_values = paths[:, -1]
        return {
            "p10": np.percentile(last_values, 10),
            "p50": np.percentile(last_values, 50),
            "p90": np.percentile(last_values, 90),
            "mean": np.mean(last_values)
        }

if __name__ == "__main__":
    sim = StochasticSimulator()
    p = sim.geometric_brownian_motion(100, 0.05, 0.2, 30)
    print(f"[Simulator] Generated {len(p)} paths for 30 days. Final P50: {sim.get_stats(p)['p50']:.2f}")
