#!/usr/bin/env python3
"""
Ingest ondata politiche 2022 election data into Aleph DuckDB.

Data sources:
  - liste/rawdata/*.json → candidate lists (metadata only, no vote counts)
  - affluenza-risultati/dati/risultati/*-comune.csv → vote counts per comune
  - affluenza-risultati/dati/risultati/*-comune_anagrafica.csv → ISTAT code mapping + elettori/votanti

Strategy:
  1. Build ISTAT→comune mapping + elettori/votanti from anagrafica CSVs
  2. Process camera-italia-comune.csv and senato-italia-comune.csv (uninominale, comune-level)
  3. Apply party canonical lookup using hardcoded mappings
  4. Insert into election_results table via DuckDB

Note: JSON files contain pre-election candidate lists (cod_ente = constituency code, NOT ISTAT).
      CSV files contain post-election actual results. Plurinominale data is at constituency level
      (CR_CP), not comune level, so it's excluded from this comune-level ingestion.
"""

import csv
import json
import os
import duckdb

# ── Configuration ──────────────────────────────────────────────────────────

DATA_DIR = "/tmp/ondata-politiche-2022"
DB_PATH = "/tmp/opencode/aleph/data/aleph.duckdb"

# Party canonical lookup (exact match on desc_lista from CSVs)
PARTY_MAP = {
    "FRATELLI D'ITALIA CON GIORGIA MELONI": "fratelli-italia",
    "LEGA PER SALVINI PREMIER": "lega",
    "FORZA ITALIA": "forza-italia",
    "PARTITO DEMOCRATICO - ITALIA DEMOCRATICA E PROGRESSISTA": "partito-democratico",
    "MOVIMENTO 5 STELLE": "movimento-5-stelle",
    "AZIONE - ITALIA VIVA - CALENDA": "azione-italia-viva",
    "ALLEANZA VERDI E SINISTRA": "verdi-sinistra",
    "+EUROPA": "piu-europa",
    "ITALIA SOVRANA E POPOLARE": "italia-sovrana-popolare",
    "UNIONE POPOLARE": "unione-popolare",
    "NOI MODERATI/LUPI - TOTI - BRUGNARO - UDC": "noi-moderati",
    "SUD CHIAMA NORD": "sud-chiama-nord",
    "VITA": "vita",
    "ITALEXIT PER L'ITALIA": "italexit",
    "IMPEGNO CIVICO LUIGI DI MAIO - CENTRO DEMOCRATICO": "impegno-civico",
    "MASTELLA NOI DI CENTRO EUROPEISTI": "noi-di-centro",
    "PARTITO COMUNISTA ITALIANO": "partito-comunista-italiano",
    "ALTERNATIVA PER L'ITALIA (APLI)": "alternativa",
    "SUSSIDIARIETA'": "sussidiarieta",
    "PARTITO ANIMALISTA - UCDL - 10 VOLTE MEGLIO": "partito-animalista",
    "PER L'ITALIA CON PARAGONE": "paragone",
    "NOI DI CENTRO - EUROPEISTI": "noi-di-centro",
    "DESTRE UNITE": "destre-unite",
    "LIBERTA'": "liberta",
}


def load_anagrafica(csv_path):
    """
    Build mapping: codice → {istat, comune, elettori, votanti}
    """
    mapping = {}
    with open(csv_path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            codice = row["codice"]
            istat = row.get("CODICE ISTAT", "").strip()
            comune = row.get("desc_com", "").strip()
            ele_t = int(row.get("ele_t", 0) or 0)
            vot_t = int(row.get("vot_t", 0) or 0)
            mapping[codice] = {
                "istat": istat,
                "comune": comune,
                "elettori": ele_t,
                "votanti": vot_t,
            }
    return mapping


def process_risultati(risultati_csv, anagrafica, election_type, year):
    """
    Process a risultati CSV, aggregate by (comune_istat, lista), return rows.
    """
    # Temporary: collect raw rows keyed by (comune_istat, lista)
    raw = {}
    anagrafica_by_istat = {}

    with open(risultati_csv, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            codice = row["codice"]
            ana = anagrafica.get(codice)
            if not ana:
                continue

            desc_lista = row["desc_lis"].strip()
            voti = int(row["voti"])
            istat = ana["istat"]

            # Build anagrafica_by_istat (once per istat)
            if istat not in anagrafica_by_istat:
                anagrafica_by_istat[istat] = ana

            # Aggregate: same (istat, lista) → sum votes
            key = (istat, desc_lista)
            if key not in raw:
                raw[key] = {"voti": 0, "perc_sum": 0.0, "perc_count": 0}
            raw[key]["voti"] += voti

            perc_str = row.get("perc", "0").replace(",", ".")
            try:
                perc = float(perc_str)
            except ValueError:
                perc = 0.0
            raw[key]["perc_sum"] += perc
            raw[key]["perc_count"] += 1

    # Transform to final rows
    rows = []
    for (istat, desc_lista), agg in raw.items():
        ana = anagrafica_by_istat.get(istat, {})
        party_canonical = PARTY_MAP.get(desc_lista, desc_lista.lower().replace(" ", "-"))
        rows.append({
            "election_type": election_type,
            "level": "comune",
            "year": year,
            "comune": ana.get("comune", ""),
            "comune_istat": istat,
            "lista": desc_lista,
            "party_canonical": party_canonical,
            "voti": agg["voti"],
            "percentuale": round(agg["perc_sum"] / agg["perc_count"], 2) if agg["perc_count"] else 0,
            "seggi": 0,
            "elettori": ana.get("elettori", 0),
            "votanti": ana.get("votanti", 0),
        })

    return rows


def main():
    print("=" * 60)
    print("Ondata politiche 2022 → Aleph DuckDB ingestion")
    print("=" * 60)

    # ── Step 1: Read JSON metadata ──────────────────────────────────────
    json_files = {
        f"{DATA_DIR}/liste/rawdata/CAMERA_ITALIA_20220925_uni.json": "Camera uninominali",
        f"{DATA_DIR}/liste/rawdata/CAMERA_ITALIA_20220925_pluri.json": "Camera plurinominali",
        f"{DATA_DIR}/liste/rawdata/SENATO_ITALIA_20220925_uni.json": "Senato uninominali",
        f"{DATA_DIR}/liste/rawdata/SENATO_ITALIA_20220925_pluri.json": "Senato plurinominali",
    }

    for path, desc in json_files.items():
        try:
            with open(path) as f:
                data = json.load(f)
                meta = data.get("metadata", {})
                print(f"  JSON [{desc}]: elez={meta.get('elez')}, dt_elez={meta.get('dt_elez')}, "
                      f"candidati={len(data.get('candidati', []))}")
        except Exception as e:
            print(f"  JSON [{desc}]: ERROR - {e}")

    # ── Step 2: Load anagrafica for Camera + Senato ─────────────────────
    print("\n  Loading anagrafica mappings...")
    camera_ana = load_anagrafica(
        f"{DATA_DIR}/affluenza-risultati/dati/risultati/camera-italia-comune_anagrafica.csv"
    )
    senato_ana = load_anagrafica(
        f"{DATA_DIR}/affluenza-risultati/dati/risultati/senato-italia-comune_anagrafica.csv"
    )
    print(f"    Camera codici: {len(camera_ana)}")
    print(f"    Senato codici: {len(senato_ana)}")

    # ── Step 3: Process vote data ────────────────────────────────────────
    print("\n  Processing vote data...")
    camera_rows = process_risultati(
        f"{DATA_DIR}/affluenza-risultati/dati/risultati/camera-italia-comune.csv",
        camera_ana,
        election_type="politiche",
        year=2022,
    )
    senato_rows = process_risultati(
        f"{DATA_DIR}/affluenza-risultati/dati/risultati/senato-italia-comune.csv",
        senato_ana,
        election_type="politiche",
        year=2022,
    )
    all_rows = camera_rows + senato_rows
    print(f"    Camera rows: {len(camera_rows)}")
    print(f"    Senato rows: {len(senato_rows)}")
    print(f"    Total rows : {len(all_rows)}")

    # ── Step 4: Check DuckDB connection ──────────────────────────────────
    print(f"\n  Connecting to DuckDB: {DB_PATH}")
    if not os.path.exists(DB_PATH):
        print(f"    ERROR: DuckDB file not found at {DB_PATH}")
        return 1
    con = duckdb.connect(DB_PATH, read_only=False)

    # Verify table exists
    tables = con.execute("SELECT name FROM sqlite_master WHERE type='table'").fetchall()
    table_names = [t[0] for t in tables]
    if "election_results" not in table_names:
        print(f"    ERROR: 'election_results' table not found in DB")
        con.close()
        return 1
    print(f"    Found 'election_results' table")

    # ── Step 5: Insert data ──────────────────────────────────────────────
    print("\n  Inserting data...")
    con.execute("BEGIN TRANSACTION")

    inserted = 0
    for i, row in enumerate(all_rows):
        con.execute("""
            INSERT INTO election_results
                (election_type, level, year, comune, comune_istat,
                 lista, party_canonical, voti, percentuale, seggi,
                 elettori, votanti)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, [
            row["election_type"],
            row["level"],
            row["year"],
            row["comune"],
            row["comune_istat"],
            row["lista"],
            row["party_canonical"],
            row["voti"],
            row["percentuale"],
            row["seggi"],
            row["elettori"],
            row["votanti"],
        ])
        inserted += 1
        if inserted % 5000 == 0:
            print(f"    ... {inserted}/{len(all_rows)} rows inserted")

    con.execute("COMMIT")
    print(f"    Done. {inserted} rows inserted.")

    # ── Step 6: Verification ─────────────────────────────────────────────
    print("\n  Verifying...")
    count = con.execute(
        "SELECT COUNT(*) FROM election_results WHERE election_type='politiche' AND year=2022"
    ).fetchone()[0]
    print(f"    SELECT COUNT(*) WHERE election_type='politiche' AND year=2022 = {count}")

    # Sample check
    sample = con.execute("""
        SELECT comune_istat, comune, lista, party_canonical, voti, elettori, votanti
        FROM election_results
        WHERE election_type='politiche' AND year=2022
        LIMIT 5
    """).fetchall()
    print("\n  Sample rows:")
    for s in sample:
        print(f"    {s}")

    con.close()
    print("\n  ✓ Ingestion complete")
    return 0


if __name__ == "__main__":
    exit(main())
