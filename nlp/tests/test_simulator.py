"""Tests for NLP simulator.py — StochasticSimulator."""
import sys
import os
import numpy as np
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from simulator import StochasticSimulator


def set_numpy_seed():
    """Ensure reproducible random results."""
    np.random.seed(42)


class TestStochasticSimulatorInit:
    def test_default_dt(self):
        sim = StochasticSimulator()
        assert sim.dt == 1 / 252

    def test_custom_dt(self):
        sim = StochasticSimulator(dt=0.01)
        assert sim.dt == 0.01


class TestGeometricBrownianMotion:
    def test_returns_2d_array(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)
        assert isinstance(paths, np.ndarray)
        assert paths.ndim == 2
        assert paths.shape == (100, 31)  # 100 sims, 31 points (T_days + 1)

    def test_default_num_sims(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)
        assert paths.shape[0] == 100

    def test_custom_num_sims(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=10, num_sims=50)
        assert paths.shape == (50, 11)

    def test_paths_start_at_S0(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=50, mu=0.05, sigma=0.2, T_days=30)
        assert np.allclose(paths[:, 0], 50.0)

    def test_all_paths_positive(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)
        assert np.all(paths > 0)

    def test_zero_volatility_gives_deterministic(self):
        set_numpy_seed()
        sim = StochasticSimulator(dt=1/252)
        # With sigma=0, geometric brownian motion becomes exp(mu * t)
        paths = sim.geometric_brownian_motion(S0=100, mu=0.1, sigma=0.0, T_days=30)
        # All paths should be identical (no randomness with sigma=0)
        assert np.allclose(paths, paths[0, :])

    def test_reproducible_with_seed(self):
        np.random.seed(123)
        sim1 = StochasticSimulator()
        paths1 = sim1.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)

        np.random.seed(123)
        sim2 = StochasticSimulator()
        paths2 = sim2.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)

        assert np.allclose(paths1, paths2)

    def test_T_days_one(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=1)
        assert paths.shape == (100, 2)  # S0 + 1 step


class TestGetStats:
    def test_returns_all_percentiles(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)
        stats = sim.get_stats(paths)
        assert "p10" in stats
        assert "p50" in stats
        assert "p90" in stats
        assert "mean" in stats

    def test_stats_are_floats(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)
        stats = sim.get_stats(paths)
        for key in ["p10", "p50", "p90", "mean"]:
            assert isinstance(stats[key], (float, np.floating))

    def test_p10_less_than_p50_less_than_p90(self):
        set_numpy_seed()
        sim = StochasticSimulator()
        paths = sim.geometric_brownian_motion(S0=100, mu=0.05, sigma=0.2, T_days=30)
        stats = sim.get_stats(paths)
        assert stats["p10"] <= stats["p50"] <= stats["p90"]

    def test_single_path_gives_nan_percentiles(self):
        sim = StochasticSimulator()
        single_path = np.array([[100, 101, 102, 103, 104]])
        stats = sim.get_stats(single_path)
        assert stats["p10"] == stats["p50"] == stats["p90"] == 104.0
        assert stats["mean"] == 104.0
