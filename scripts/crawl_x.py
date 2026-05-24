#!/tmp/aleph_venv/bin/python3
"""
X/Twitter crawler for the Aleph political analysis project.

Fetches recent tweets from Italian politicians and stores them in DuckDB.

Configuration (env vars):
    X_BEARER_TOKEN      — Twitter API v2 Bearer Token (required)
    ALEPH_DB            — Path to DuckDB database (default: data/aleph.duckdb)
    X_RATE_LIMIT_SLEEP  — Seconds to wait between API calls (default: 2)
    X_LOOKBACK_HOURS    — How many hours back to fetch (default: 24)

Usage:
    export X_BEARER_TOKEN='AAAAAAAA...'
    python scripts/crawl_x.py

Database tables created/used:
    politici            — Source table with politician handles
    posts_x             — Collected tweets
"""

import os
import time
import logging
from datetime import datetime, timedelta, UTC

import duckdb
import tweepy

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
log = logging.getLogger("crawl_x")

DB_PATH = os.environ.get("ALEPH_DB", "data/aleph.duckdb")
BEARER_TOKEN = os.environ.get("X_BEARER_TOKEN")
RATE_LIMIT_SLEEP = float(os.environ.get("X_RATE_LIMIT_SLEEP", "2"))
LOOKBACK_HOURS = int(os.environ.get("X_LOOKBACK_HOURS", "24"))


def setup_tables(con: duckdb.DuckDBPyConnection) -> None:
    """Create politici and posts_x tables if they don't exist."""
    con.execute("""
        CREATE TABLE IF NOT EXISTS politici (
            id          INTEGER PRIMARY KEY,
            full_name   VARCHAR NOT NULL,
            party       VARCHAR,
            screen_name_x VARCHAR,
            username_ig   VARCHAR,
            page_id_fb    VARCHAR
        )
    """)
    con.execute("""
        CREATE TABLE IF NOT EXISTS posts_x (
            id              INTEGER PRIMARY KEY,
            politico_id     INTEGER NOT NULL,
            tweet_id        VARCHAR UNIQUE,
            content         VARCHAR,
            posted_at       TIMESTAMP,
            url             VARCHAR,
            retweet_count   INTEGER DEFAULT 0,
            reply_count     INTEGER DEFAULT 0,
            like_count      INTEGER DEFAULT 0,
            quote_count     INTEGER DEFAULT 0,
            fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (politico_id) REFERENCES politici(id)
        )
    """)


def get_politicians(con: duckdb.DuckDBPyConnection) -> list[tuple]:
    """Fetch politicians with X/Twitter handles from the politici table.

    Returns:
        List of (id, full_name, party, screen_name_x) tuples.
    """
    return con.execute("""
        SELECT id, full_name, party, screen_name_x
        FROM politici
        WHERE screen_name_x IS NOT NULL AND screen_name_x != ''
        ORDER BY full_name
    """).fetchall()


def fetch_tweets_for_user(
    handle: str, since: datetime
) -> list[dict]:
    """Fetch recent tweets for a single Twitter user.

    Args:
        handle: Twitter screen name (without @).
        since: Fetch tweets posted after this datetime.

    Returns:
        List of tweet dicts with keys: tweet_id, content, posted_at,
        retweet_count, reply_count, like_count, quote_count.

    NOTE: This is a PLACEHOLDER. Replace with actual Twitter API v2 calls.
          Use tweepy.Client or requests to:
            POST https://api.twitter.com/2/tweets/search/recent
          with query ``from:{handle} -is:retweet`` and the Bearer Token.
    """
    if not BEARER_TOKEN:
        log.warning("No Bearer Token — skipping @%s", handle)
        return []

    client = tweepy.Client(
        bearer_token=BEARER_TOKEN,
        wait_on_rate_limit=True,
    )

    start_time = since.isoformat().replace("+00:00", "Z")
    query = f"from:{handle} -is:retweet"

    all_tweets: list[dict] = []
    pagination_token: str | None = None

    MAX_PAGES = 10
    for page_num in range(1, MAX_PAGES + 1):
        try:
            response = client.search_recent_tweets(
                query=query,
                start_time=start_time,
                tweet_fields=["created_at", "public_metrics"],
                max_results=100,
                pagination_token=pagination_token,
            )
        except tweepy.TweepyException as exc:
            log.warning("API error for @%s (page %d): %s", handle, page_num, exc)
            break

        if response.data:
            for t in response.data:
                metrics = t.public_metrics or {}
                all_tweets.append({
                    "tweet_id": t.id,
                    "content": t.text,
                    "posted_at": t.created_at,
                    "retweet_count": metrics.get("retweet_count", 0),
                    "reply_count": metrics.get("reply_count", 0),
                    "like_count": metrics.get("like_count", 0),
                    "quote_count": metrics.get("quote_count", 0),
                })

        if response.meta is None or response.meta.get("next_token") is None:
            break
        pagination_token = response.meta["next_token"]

    if all_tweets:
        log.debug("Fetched %d tweets for @%s", len(all_tweets), handle)
    return all_tweets


def store_tweets(
    con: duckdb.DuckDBPyConnection,
    politico_id: int,
    handle: str,
    tweets: list[dict],
) -> int:
    """Insert fetched tweets into the posts_x table, skipping duplicates.

    Returns:
        Number of new tweets inserted.
    """
    inserted = 0
    for t in tweets:
        try:
            con.execute(
                """
                INSERT OR IGNORE INTO posts_x
                    (politico_id, tweet_id, content, posted_at, url,
                     retweet_count, reply_count, like_count, quote_count)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    politico_id,
                    t["tweet_id"],
                    t["content"],
                    t["posted_at"],
                    f"https://twitter.com/{handle}/status/{t['tweet_id']}",
                    t.get("retweet_count", 0),
                    t.get("reply_count", 0),
                    t.get("like_count", 0),
                    t.get("quote_count", 0),
                ),
            )
            inserted += con.execute("SELECT changes()").fetchone()[0]
        except Exception:
            log.exception("Failed to insert tweet %s for @%s", t.get("tweet_id"), handle)
    return inserted


def fetch_and_store(
    con: duckdb.DuckDBPyConnection,
    politici: list[tuple],
    since: datetime,
) -> dict[str, int]:
    """Fetch tweets for each politician and store in DuckDB.

    Args:
        con: DuckDB connection.
        politici: List of (id, full_name, party, screen_name_x) tuples.
        since: Fetch tweets posted after this datetime.

    Returns:
        Dict with stats: {"fetched": N, "stored": N, "errors": N}.
    """
    stats = {"fetched": 0, "stored": 0, "errors": 0}
    total = len(politici)

    for idx, (pid, name, party, handle) in enumerate(politici, start=1):
        log.info("[%d/%d] @%s (%s — %s)", idx, total, handle, name, party)
        stats["fetched"] += 1

        try:
            tweets = fetch_tweets_for_user(handle, since)
            stored = store_tweets(con, pid, handle, tweets)
            stats["stored"] += stored
            log.info("  Stored %d new tweets for @%s", stored, handle)
        except Exception:
            log.exception("  Error fetching @%s", handle)
            stats["errors"] += 1

        # Rate-limit sleep between users
        time.sleep(RATE_LIMIT_SLEEP)

    return stats


def main() -> None:
    """Entry point for the X/Twitter crawler."""
    log.info("=== X/Twitter Crawler ===")

    if not BEARER_TOKEN:
        log.error(
            "X_BEARER_TOKEN not set. Cannot fetch tweets. "
            "Set the X_BEARER_TOKEN environment variable to run the crawler."
        )

    con = duckdb.connect(DB_PATH)
    setup_tables(con)

    since = datetime.now(UTC) - timedelta(hours=LOOKBACK_HOURS)
    log.info("Fetching tweets since %s (lookback: %dh)", since.isoformat(), LOOKBACK_HOURS)

    politici = get_politicians(con)
    if not politici:
        log.warning("No politicians with X handles found in politici table.")
        con.close()
        return

    log.info("Found %d politicians with X handles", len(politici))
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
