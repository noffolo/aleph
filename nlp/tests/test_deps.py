"""Dependency verification tests for NLP sidecar."""


def test_cmdstanpy_installed():
    """cmdstanpy is required by Prophet for Bayesian forecasting.
    Without it, Prophet crashes at fit-time."""
    import cmdstanpy  # noqa: F401
