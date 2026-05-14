"""Integration tests for NLP main.py — DuckDB, StreamPredictions, serve."""
import sys
import os
import json
import tempfile
import numpy as np
import pandas as pd
import duckdb
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

import nlp_pb2
from main import load_history_from_duckdb, NLPService, _check_identifier


def _temp_db_path():
    fd, path = tempfile.mkstemp(suffix='.duckdb')
    os.close(fd)
    os.unlink(path)
    return path


class TestLoadHistoryFromDuckDBIntegration:
    def test_loads_real_data(self):
        dbpath = _temp_db_path()
        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE test_data (my_date DATE, my_value DOUBLE)")
            dates = pd.date_range('2024-01-01', periods=20, freq='D')
            for i, d in enumerate(dates):
                con.execute("INSERT INTO test_data VALUES (?, ?)",
                            [d.strftime('%Y-%m-%d'), float(i * 10 + 100)])
            con.close()

            df = load_history_from_duckdb("test_data", dbpath)
            assert df is not None
            assert len(df) == 20
            assert 'ds' in df.columns
            assert 'y' in df.columns
            assert df['y'].iloc[-1] == 290.0
        finally:
            os.unlink(dbpath)

    def test_returns_none_for_nonexistent_table(self):
        dbpath = _temp_db_path()

        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE other_table (ds DATE, y DOUBLE)")
            con.close()

            df = load_history_from_duckdb("nonexistent_table", dbpath)
            assert df is None
        finally:
            os.unlink(dbpath)

    def test_returns_none_for_too_few_rows(self):
        dbpath = _temp_db_path()

        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE small_data (ts TIMESTAMP, val FLOAT)")
            for i in range(5):
                con.execute("INSERT INTO small_data VALUES (?, ?)",
                            [f'2024-01-{i+1:02d} 00:00:00', float(i)])
            con.close()

            df = load_history_from_duckdb("small_data", dbpath)
            assert df is None
        finally:
            os.unlink(dbpath)

    def test_handles_timestamp_columns(self):
        dbpath = _temp_db_path()

        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE ts_data (created_at TIMESTAMP, metric BIGINT)")
            for i in range(15):
                con.execute("INSERT INTO ts_data VALUES (?, ?)",
                            [f'2024-01-{i+1:02d} 00:00:00', i * 100])
            con.close()

            df = load_history_from_duckdb("ts_data", dbpath)
            assert df is not None
            assert len(df) >= 15
        finally:
            os.unlink(dbpath)

    def test_handles_integer_value_column(self):
        dbpath = _temp_db_path()

        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE int_data (event_date DATE, count INTEGER)")
            for i in range(12):
                con.execute("INSERT INTO int_data VALUES (?, ?)",
                            [f'2024-{i+1:02d}-01', i * 10])
            con.close()

            df = load_history_from_duckdb("int_data", dbpath)
            assert df is not None
            assert len(df) == 12
        finally:
            os.unlink(dbpath)

    def test_handles_decimal_column(self):
        dbpath = _temp_db_path()

        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE dec_data (dt DATE, price DECIMAL(10,2))")
            for i in range(15):
                con.execute("INSERT INTO dec_data VALUES (?, ?)",
                            [f'2024-01-{i+1:02d}', float(i * 15.5)])
            con.close()

            df = load_history_from_duckdb("dec_data", dbpath)
            assert df is not None
        finally:
            os.unlink(dbpath)

    def test_requires_date_and_value_columns(self):
        dbpath = _temp_db_path()

        try:
            con = duckdb.connect(dbpath)
            con.execute("CREATE TABLE text_only (id INTEGER, name VARCHAR)")
            con.execute("INSERT INTO text_only VALUES (1, 'hello')")
            con.close()

            df = load_history_from_duckdb("text_only", dbpath)
            assert df is None
        finally:
            os.unlink(dbpath)


class TestNLPServiceStreamPredictions:
    def _make_service(self):
        svc = NLPService.__new__(NLPService)
        svc.duckdb_path = ""
        svc.markets = None
        svc.ensemble = None
        svc.simulator = None
        return svc

    def test_no_duckdb_data_returns_not_found(self):
        import pytest
        pytest.skip("Requires full NLPService with wired markets/ensemble")

    def test_cancelled_before_market_returns_early(self):
        svc = self._make_service()
        req = nlp_pb2.StreamPredictionsRequest(context_id="test", ontology_query="")

        class Ctx:
            def __init__(self):
                self.calls = 0
            def set_code(self, code): pass
            def set_details(self, details): pass
            def is_active(self):
                self.calls += 1
                return self.calls < 2

        ctx = Ctx()
        result = svc.StreamPredictions(req, ctx)
        output = list(result) if hasattr(result, '__iter__') else []
        assert output == []

    def test_no_data_with_empty_context_id(self):
        svc = self._make_service()
        svc.duckdb_path = "/nonexistent.duckdb"
        req = nlp_pb2.StreamPredictionsRequest(context_id="", ontology_query="")

        class Ctx:
            def __init__(self):
                self.code = None
            def set_code(self, code):
                self.code = code
            def set_details(self, details): pass
            def is_active(self):
                return True

        ctx = Ctx()
        svc.StreamPredictions(req, ctx)
