#!/tmp/aleph_venv/bin/python3
"""
Instagram crawler for the Aleph political analysis project.

Fetches recent Instagram posts from Italian politicians and stores them in DuckDB.

Configuration (env vars):
    IG_USERNAME         — Instagram account username (required)
    IG_PASSWORD         — Instagram account password (required)
    ALEPH_DB            — Path to DuckDB database (default: data/aleph.duckdb)
    IG_RATE_LIMIT_SLEEP — Seconds to wait between API calls (default: 3)
    IG_LOOKBACK_HOURS   — How many hours back to fetch (default: 24)
    IG_POSTS_PER_USER   — Max posts to fetch per user (default: 20)

Usage:
    export IG_USERNAME='my_bot_account' IG_PASSWORD='secret'
    python scripts/crawl_instagram.py

Database tables created/used:
    politici            — Source table with politician handles
    posts_ig            — Collected Instagram posts
"""

import os
import time
import logging
from datetime import datetime, timedelta, UTC

import duckdb
import instaloader

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
log = logging.getLogger("crawl_ig")

DB_PATH = os.environ.get("ALEPH_DB", "data/aleph.duckdb")
IG_USERNAME = os.environ.get("IG_USERNAME")
IG_PASSWORD = os.environ.get("IG_PASSWORD")
RATE_LIMIT_SLEEP = float(os.environ.get("IG_RATE_LIMIT_SLEEP", "3"))
LOOKBACK_HOURS = int(os.environ.get("IG_LOOKBACK_HOURS", "24"))
POSTS_PER_USER = int(os.environ.get("IG_POSTS_PER_USER", "20"))


def setup_tables(con: duckdb.DuckDBPyConnection) -> None:
    """Create politici and posts_ig tables if they don't exist."""
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
        CREATE TABLE IF NOT EXISTS posts_ig (
            id            INTEGER PRIMARY KEY,
            politico_id   INTEGER NOT NULL,
            post_id       VARCHAR UNIQUE,
            caption       VARCHAR,
            media_type    VARCHAR,
            image_url     VARCHAR,
            post_url      VARCHAR,
            posted_at     TIMESTAMP,
            like_count    INTEGER DEFAULT 0,
            comment_count INTEGER DEFAULT 0,
            fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (politico_id) REFERENCES politici(id)
        )
    """)


def get_politicians(con: duckdb.DuckDBPyConnection) -> list[tuple]:
    """Fetch politicians with Instagram usernames from the politici table."""
    return con.execute("""
        SELECT id, full_name, party, username_ig
        FROM politici
        WHERE username_ig IS NOT NULL AND username_ig != ''
        ORDER BY full_name
    """).fetchall()


def fetch_posts_for_user(username: str, since: datetime) -> list[dict]:
    """Fetch recent Instagram posts for a single user via Instaloader.

    Uses an Instaloader session file for efficient, rate-limit-friendly login.
    Creates the session file on first run if it doesn't exist.

    Returns:
        List of post dicts with keys: post_id, caption, media_type,
        image_url, post_url, posted_at, like_count, comment_count.
    """
    if not IG_USERNAME or not IG_PASSWORD:
        log.warning("IG_USERNAME/IG_PASSWORD not set — skipping @%s", username)
        return []

    L = instaloader.Instaloader(sleep=True, quiet=True)

    # Login with session file for efficiency (creates session file if missing)
    session_dir = os.path.expanduser("~/.config/aleph")
    os.makedirs(session_dir, exist_ok=True)
    session_file = os.path.join(session_dir, "instaloader.session")

    try:
        L.load_session_from_file(IG_USERNAME, session_file)
        log.debug("Loaded existing Instaloader session for %s", IG_USERNAME)
    except FileNotFoundError:
        log.info("No session file found — logging in and saving session for %s", IG_USERNAME)
        L.login(IG_USERNAME, IG_PASSWORD)
        L.save_session_to_file(session_file)

    try:
        profile = instaloader.Profile.from_username(L.context, username)
    except instaloader.exceptions.ProfileNotExistsException:
        log.warning("Profile @%s does not exist or is not accessible", username)
        return []
    except Exception:
        log.exception("Failed to load profile for @%s", username)
        return []

    results: list[dict] = []
    for post in profile.get_posts():
        post_date = post.date_local
        if post_date.tzinfo is not None:
            post_date = post_date.replace(tzinfo=None)
        if post_date < since:
            break
        results.append({
            "post_id": str(post.mediaid),
            "caption": post.caption or "",
            "media_type": "VIDEO" if post.is_video else "IMAGE",
            "image_url": post.url,
            "post_url": f"https://instagram.com/p/{post.shortcode}/",
            "posted_at": post.date_local,
            "like_count": post.likes,
            "comment_count": post.comments,
        })
        if len(results) >= POSTS_PER_USER:
            break

    return results


def store_posts(
    con: duckdb.DuckDBPyConnection,
    politico_id: int,
    posts: list[dict],
) -> int:
    """Insert fetched posts into the posts_ig table, skipping duplicates."""
    inserted = 0
    for p in posts:
        try:
            con.execute(
                """
                INSERT OR IGNORE INTO posts_ig
                    (politico_id, post_id, caption, media_type, image_url,
                     post_url, posted_at, like_count, comment_count)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    politico_id,
                    p["post_id"],
                    p["caption"],
                    p["media_type"],
                    p["image_url"],
                    p["post_url"],
                    p["posted_at"],
                    p.get("like_count", 0),
                    p.get("comment_count", 0),
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
    """Fetch Instagram posts for each politician and store in DuckDB."""
    stats = {"fetched": 0, "stored": 0, "errors": 0}
    total = len(politici)

    for idx, (pid, name, party, username) in enumerate(politici, start=1):
        log.info("[%d/%d] @%s (%s — %s)", idx, total, username, name, party)
        stats["fetched"] += 1

        try:
            posts = fetch_posts_for_user(username, since)
            stored = store_posts(con, pid, posts)
            stats["stored"] += stored
            log.info("  Stored %d new posts for @%s", stored, username)
        except Exception:
            log.exception("  Error fetching @%s", username)
            stats["errors"] += 1

        time.sleep(RATE_LIMIT_SLEEP)

    return stats


def main() -> None:
    """Entry point for the Instagram crawler."""
    log.info("=== Instagram Crawler ===")

    if not IG_USERNAME or not IG_PASSWORD:
        log.warning(
            "IG_USERNAME and/or IG_PASSWORD not set. "
            "Running in DRY-RUN mode — no data will be fetched."
        )

    con = duckdb.connect(DB_PATH)
    setup_tables(con)

    since = datetime.now(UTC) - timedelta(hours=LOOKBACK_HOURS)
    log.info("Fetching posts since %s (lookback: %dh)", since.isoformat(), LOOKBACK_HOURS)

    politici = get_politicians(con)
    if not politici:
        log.warning("No politicians with Instagram usernames found in politici table.")
        con.close()
        return

    log.info("Found %d politicians with Instagram handles", len(politici))
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
