#!/tmp/aleph_venv/bin/python3
import os, sys, time, logging, asyncio
from datetime import datetime, timedelta, timezone

import duckdb
import yaml

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
log = logging.getLogger("crawl_telegram")

DB_PATH = os.environ.get("ALEPH_DB", "data/aleph.duckdb")
CHANNELS_YAML = os.environ.get("TELEGRAM_CHANNELS_YAML", "scripts/telegram_channels.yaml")
API_ID = int(os.environ.get("TELEGRAM_API_ID", "35222533"))
API_HASH = os.environ.get("TELEGRAM_API_HASH", "ccde85ec5fe5c175cea50303469cf05f")
PHONE = os.environ.get("TELEGRAM_PHONE", "+393400816352")
SESSION_PATH = os.path.expanduser(os.environ.get("TELEGRAM_SESSION", "~/.config/aleph/telegram.session"))
MESSAGES_PER_CHANNEL = int(os.environ.get("TELEGRAM_MESSAGES_PER_CHANNEL", "50"))
CONTINUOUS = os.environ.get("TELEGRAM_CONTINUOUS", "0") == "1"
POLL_INTERVAL = int(os.environ.get("TELEGRAM_POLL_INTERVAL_MIN", "15"))


def load_channels(path: str) -> list[dict]:
    with open(path) as f:
        data = yaml.safe_load(f)
    return [c for c in data.get("channels", []) if c.get("status") == "ACTIVE"]


def create_tables(con: duckdb.DuckDBPyConnection) -> None:
    con.execute("""
        CREATE TABLE IF NOT EXISTS posts_telegram (
            id           INTEGER PRIMARY KEY,
            channel_name VARCHAR,
            message_id   VARCHAR UNIQUE,
            content      VARCHAR,
            has_media    BOOLEAN DEFAULT FALSE,
            posted_at    TIMESTAMP,
            fetched_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    con.execute("""
        CREATE TABLE IF NOT EXISTS crawl_watermarks (
            channel_name   VARCHAR PRIMARY KEY,
            last_message_id VARCHAR,
            last_posted_at TIMESTAMP,
            fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    con.execute("CREATE INDEX IF NOT EXISTS idx_tg_channel ON posts_telegram(channel_name)")
    con.execute("CREATE INDEX IF NOT EXISTS idx_tg_posted ON posts_telegram(posted_at)")


def get_watermark(con: duckdb.DuckDBPyConnection, channel: str) -> datetime | None:
    row = con.execute(
        "SELECT last_posted_at FROM crawl_watermarks WHERE channel_name=?", (channel,)
    ).fetchone()
    return row[0] if row and row[0] else None


def set_watermark(con: duckdb.DuckDBPyConnection, channel: str, msg_id: str, posted_at: datetime) -> None:
    now_val = datetime.now(timezone.utc)
    con.execute("""
        INSERT INTO crawl_watermarks (channel_name, last_message_id, last_posted_at, fetched_at)
        VALUES (?,?,?,?)
        ON CONFLICT(channel_name) DO UPDATE SET
            last_message_id=EXCLUDED.last_message_id,
            last_posted_at=EXCLUDED.last_posted_at,
            fetched_at=EXCLUDED.fetched_at
    """, (channel, msg_id, posted_at, now_val))


async def crawl_async(channels: list[dict], db_path: str) -> dict:
    from telethon import TelegramClient, errors

    os.makedirs(os.path.dirname(SESSION_PATH), exist_ok=True)
    client = TelegramClient(SESSION_PATH, API_ID, API_HASH, system_version="4.16.32")
    await client.start(phone=PHONE)

    con = duckdb.connect(db_path)
    create_tables(con)

    stats = {"stored": 0, "channels": 0, "errors": 0}

    for ch in channels:
        uname = ch["username"]
        title = ch.get("title", uname)
        since = get_watermark(con, uname)
        limit = MESSAGES_PER_CHANNEL

        try:
            entity = await client.get_entity(uname)
            kwargs: dict = {"limit": limit}
            if since:
                kwargs["offset_date"] = since
            msgs = await client.get_messages(entity, **kwargs)
        except errors.FloodWaitError as e:
            log.warning("FloodWait @%s: %ds", uname, e.seconds)
            await asyncio.sleep(e.seconds)
            continue
        except (errors.ChannelPrivateError, errors.ChannelInvalidError, errors.UsernameNotOccupiedError) as e:
            log.warning("Skip @%s: %s", uname, e)
            continue
        except Exception:
            log.exception("Error fetching @%s", uname)
            stats["errors"] += 1
            continue

        n = 0
        for m in msgs:
            if not m.text or len(m.text.strip()) < 10:
                continue
            try:
                con.execute("""
                    INSERT OR IGNORE INTO posts_telegram (id, channel_name, message_id, content, has_media, posted_at)
                    VALUES ((SELECT COALESCE(MAX(id),0)+1 FROM posts_telegram), ?, ?, ?, ?, ?)
                """, (uname, str(m.id), m.text[:2000], m.media is not None,
                      m.date.replace(tzinfo=timezone.utc) if m.date.tzinfo else m.date))
                n += 1
            except Exception:
                log.exception("DB insert error @%s msg %s", uname, m.id)

        if n > 0:
            ts = msgs[0].date
            set_watermark(con, uname, str(msgs[0].id), ts.replace(tzinfo=timezone.utc) if ts.tzinfo else ts)
        stats["stored"] += n
        stats["channels"] += 1
        log.info("  @%s (%s): %d msg", uname, title, n)
        await asyncio.sleep(1)

    total = con.execute("SELECT COUNT(*) FROM posts_telegram").fetchone()[0]
    log.info("Cycle: %d canali, %d nuovi msg, %d totale DB, %d errori",
             stats["channels"], stats["stored"], total, stats["errors"])
    con.close()
    await client.disconnect()
    return stats


def main():
    log.info("=== Telegram Crawler ===")
    channels = load_channels(CHANNELS_YAML)
    log.info("Loaded %d channels from %s", len(channels), CHANNELS_YAML)

    if CONTINUOUS:
        log.info("Continuous mode: polling every %d min", POLL_INTERVAL)
        while True:
            stats = asyncio.run(crawl_async(channels, DB_PATH))
            if stats["stored"] == 0 and stats["errors"] == 0:
                log.info("Sleeping %d min... (no new data)", POLL_INTERVAL)
            else:
                log.info("Sleeping %d min...", POLL_INTERVAL)
            time.sleep(POLL_INTERVAL * 60)
    else:
        asyncio.run(crawl_async(channels, DB_PATH))


if __name__ == "__main__":
    main()
