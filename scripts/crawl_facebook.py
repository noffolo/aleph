#!/tmp/aleph_venv/bin/python3
"""
Facebook crawler for the Aleph political analysis project.

Fetches recent Facebook posts from Italian politician pages and stores them in DuckDB.

Configuration (env vars):
    FB_ACCESS_TOKEN     — Facebook Graph API access token (required)
    ALEPH_DB            — Path to DuckDB database (default: data/aleph.duckdb)
    FB_RATE_LIMIT_SLEEP — Seconds to wait between API calls (default: 2)
    FB_LOOKBACK_HOURS   — How many hours back to fetch (default: 24)
    FB_POSTS_PER_PAGE   — Max posts to fetch per page (default: 25)

Usage:
    export FB_ACCESS_TOKEN='EAAB...'
    python scripts/crawl_facebook.py

Database tables created/used:
    politici            — Source table with politician page IDs
    posts_fb            — Collected Facebook posts
"""

import os
import time
import logging
from datetime import datetime, timedelta, UTC

import duckdb

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
log = logging.getLogger("crawl_fb")

DB_PATH = os.environ.get("ALEPH_DB", "data/aleph.duckdb")
ACCESS_TOKEN = os.environ.get("FB_ACCESS_TOKEN")
RATE_LIMIT_SLEEP = float(os.environ.get("FB_RATE_LIMIT_SLEEP", "2"))
LOOKBACK_HOURS = int(os.environ.get("FB_LOOKBACK_HOURS", "24"))
POSTS_PER_PAGE = int(os.environ.get("FB_POSTS_PER_PAGE", "25"))


def setup_tables(con: duckdb.DuckDBPyConnection) -> None:
    """Create politici and posts_fb tables if they don't exist."""
    con.execute("""
        CREATE TABLE IF NOT EXISTS politici (
            id           INTEGER PRIMARY KEY,
            full_name    VARCHAR NOT NULL,
            party        VARCHAR,
            screen_name_x VARCHAR,
            username_ig  VARCHAR,
            page_id_fb   VARCHAR
        )
    """)
    con.execute("""
        CREATE TABLE IF NOT EXISTS posts_fb (
            id            INTEGER PRIMARY KEY,
            politico_id   INTEGER NOT NULL,
            post_id       VARCHAR UNIQUE,
            content       VARCHAR,
            post_url      VARCHAR,
            posted_at     TIMESTAMP,
            share_count   INTEGER DEFAULT 0,
            comment_count INTEGER DEFAULT 0,
            reaction_count INTEGER DEFAULT 0,
            fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (politico_id) REFERENCES politici(id)
        )
    """)


def get_politicians(con: duckdb.DuckDBPyConnection) -> list[tuple]:
    """Fetch politicians with Facebook page IDs from the politici table."""
    return con.execute("""
        SELECT id, full_name, party, page_id_fb
        FROM politici
        WHERE page_id_fb IS NOT NULL AND page_id_fb != ''
        ORDER BY full_name
    """).fetchall()


def fetch_posts_for_page(page_id: str, since: datetime) -> list[dict]:
    """Fetch recent posts from a Facebook page.

    NOTE: This is a PLACEHOLDER. Replace with actual Facebook Graph API calls.

    Recommended approach using ``requests``:
        import requests
        url = f"https://graph.facebook.com/v19.0/{page_id}/posts"
        params = {
            "access_token": ACCESS_TOKEN,
            "fields": "id,message,created_time,shares,permalink_url",
            "limit": POSTS_PER_PAGE,
            "since": since.isoformat(),
        }
        resp = requests.get(url, params=params)
        resp.raise_for_status()
        ...
    For reactions/comments you may need separate calls to /{post_id}/reactions
    and /{post_id}/comments with summary=true.

    Returns:
        List of post dicts with keys: post_id, content, post_url, posted_at,
        share_count, comment_count, reaction_count.
    """
    # ────── PLACEHOLDER: Replace with real API call ──────
    # import requests
    #
    # url = f"https://graph.facebook.com/v19.0/{page_id}/posts"
    # params = {
    #     "access_token": ACCESS_TOKEN,
    #     "fields": "id,message,created_time,shares,permalink_url",
    #     "limit": POSTS_PER_PAGE,
    #     "since": since.strftime("%Y-%m-%dT%H:%M:%S"),
    # }
    # resp = requests.get(url, params=params, timeout=30)
    # resp.raise_for_status()
    # data = resp.json()
    #
    # results = []
    # for item in data.get("data", []):
    #     results.append({
    #         "post_id": item["id"],
    #         "content": item.get("message", ""),
    #         "post_url": item.get("permalink_url", f"https://facebook.com/{item['id']}"),
    #         "posted_at": item.get("created_time"),
    #         "share_count": item.get("shares", {}).get("count", 0),
    #         "comment_count": 0,  # Requires separate /{post_id}/comments call
    #         "reaction_count": 0, # Requires separate /{post_id}/reactions call
    #     })
    # return results
    # ────── END PLACEHOLDER ─────────────────────────────

    log.debug("DRY RUN — would fetch Facebook posts for page %s", page_id)
    return []


def store_posts(
    con: duckdb.DuckDBPyConnection,
    politico_id: int,
    posts: list[dict],
) -> int:
    """Insert fetched posts into the posts_fb table, skipping duplicates."""
    inserted = 0
    for p in posts:
        try:
            con.execute(
                """
                INSERT OR IGNORE INTO posts_fb
                    (politico_id, post_id, content, post_url, posted_at,
                     share_count, comment_count, reaction_count)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    politico_id,
                    p["post_id"],
                    p["content"],
                    p["post_url"],
                    p["posted_at"],
                    p.get("share_count", 0),
                    p.get("comment_count", 0),
                    p.get("reaction_count", 0),
                ),
            )
            inserted += con.execute("SELECT changes()").fetchone()[0]
        except Exception:
            log.exception("Failed to insert post %s", p.get("post_id"))
    return inserted


def fetch_and_store(
    con: duckdb.DuckDBPyConnection,
    politici: list[tuple],
    since: datetime,
) -> dict[str, int]:
    """Fetch Facebook posts for each politician page and store in DuckDB."""
    stats = {"fetched": 0, "stored": 0, "errors": 0}
    total = len(politici)

    for idx, (pid, name, party, page_id) in enumerate(politici, start=1):
        log.info("[%d/%d] %s (%s) [%s]", idx, total, name, party, page_id)
        stats["fetched"] += 1

        try:
            posts = fetch_posts_for_page(page_id, since)
            stored = store_posts(con, pid, posts)
            stats["stored"] += stored
            log.info("  Stored %d new posts for %s", stored, name)
        except Exception:
            log.exception("  Error fetching page %s", page_id)
            stats["errors"] += 1

        time.sleep(RATE_LIMIT_SLEEP)

    return stats


def main() -> None:
    """Entry point for the Facebook crawler."""
    log.info("=== Facebook Crawler ===")

    if not ACCESS_TOKEN:
        log.warning(
            "FB_ACCESS_TOKEN not set. Running in DRY-RUN mode — "
            "no data will be fetched. Set the env var to fetch real posts."
        )

    con = duckdb.connect(DB_PATH)
    setup_tables(con)

    since = datetime.now(UTC) - timedelta(hours=LOOKBACK_HOURS)
    log.info("Fetching posts since %s (lookback: %dh)", since.isoformat(), LOOKBACK_HOURS)

    politici = get_politicians(con)
    if not politici:
        log.warning("No politicians with Facebook page IDs found in politici table.")
        con.close()
        return

    log.info("Found %d politicians with Facebook pages", len(politici))
    stats = fetch_and_store(con, politici, since)
    log.info(
        "Done. Fetched: %d, Stored: %d, Errors: %d",
        stats["fetched"],
        stats["stored"],
        stats["errors"],
    )

    con.close()


if __name__ == "__main__":
    main()
