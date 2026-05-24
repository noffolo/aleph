#!/tmp/aleph_venv/bin/python3
"""Run all social media crawlers sequentially with rate-limit pauses between platforms."""

import subprocess
import sys
import time
import logging
from pathlib import Path

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] crawl_all: %(message)s",
)
log = logging.getLogger("crawl_all")

SCRIPTS_DIR = Path(__file__).resolve().parent
PLATFORM_SCRIPTS = ["crawl_x.py", "crawl_instagram.py", "crawl_facebook.py", "crawl_telegram.py"]
INTER_PLATFORM_SLEEP = int(subprocess.os.environ.get("CRAWL_INTER_PLATFORM_SLEEP", "5"))


def run_script(script_name: str) -> bool:
    """Run a single crawler script and report its outcome."""
    script_path = SCRIPTS_DIR / script_name
    if not script_path.exists():
        log.warning("Script not found: %s", script_path)
        return False

    log.info("── Running %s ──", script_name)
    result = subprocess.run(
        [sys.executable, str(script_path)],
        capture_output=True,
        text=True,
        timeout=600,
    )

    if result.stdout:
        sys.stdout.write(result.stdout)
    if result.stderr:
        sys.stderr.write(result.stderr)

    if result.returncode != 0:
        log.error("%s exited with code %d", script_name, result.returncode)
        return False

    log.info("%s completed successfully", script_name)
    return True


def main() -> None:
    """Run all platform crawlers, pausing between each to respect rate limits."""
    log.info("=== Aleph Social Media Crawler ===")
    log.info("Platforms: %s", ", ".join(PLATFORM_SCRIPTS))

    results: dict[str, bool] = {}

    for i, script in enumerate(PLATFORM_SCRIPTS):
        results[script] = run_script(script)

        if i < len(PLATFORM_SCRIPTS) - 1:
            log.info("Pausing %ds before next platform...", INTER_PLATFORM_SLEEP)
            time.sleep(INTER_PLATFORM_SLEEP)

    passed = sum(results.values())
    total = len(results)
    log.info("=== Done: %d/%d platforms succeeded ===", passed, total)

    if passed < total:
        failed = [k for k, v in results.items() if not v]
        log.error("Failed: %s", ", ".join(failed))
        sys.exit(1)


if __name__ == "__main__":
    main()
