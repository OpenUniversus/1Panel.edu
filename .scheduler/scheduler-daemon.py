#!/usr/bin/env python3
"""1Panel.edu scheduler daemon.
Pure Python. Single long-running process. No LLM, no admin, no schtasks.
- Master tick at :00 of every 5-hour bucket (0, 5, 10, 15, 20): runs gen-plan + run-next-task
- Sub tick every 10 minutes: runs run-next-task
- Anti-re-run: state.json tracks per-task status
- Anti-miss: state.json detects window expiry
- Crash-safe: re-loads state on each tick

Run: python scheduler-daemon.py
Stop: Ctrl-C
"""
import logging
import subprocess
import sys
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
LOG_DIR       = SCHEDULER_DIR / "logs"
LOG_DIR.mkdir(exist_ok=True)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.FileHandler(LOG_DIR / "daemon.log", encoding="utf-8"),
        logging.StreamHandler(sys.stdout),
    ],
)
log = logging.getLogger("scheduler")

TZ = timezone(timedelta(hours=8))


def now_local():
    return datetime.now(TZ)


def current_bucket(now):
    """5-hour bucket: 0-4 / 5-9 / 10-14 / 15-19 / 20-23."""
    return (now.hour // 5) * 5


def seconds_until(target: datetime) -> float:
    delta = (target - now_local()).total_seconds()
    return max(0.0, delta)


def run_script(name, args=None):
    """Run a Python script in scheduler dir, return exit code."""
    script = SCHEDULER_DIR / name
    if not script.exists():
        log.error(f"missing script: {name}")
        return 1
    cmd = [sys.executable, str(script)]
    if args:
        cmd.extend(args)
    try:
        r = subprocess.run(cmd, cwd=str(SCHEDULER_DIR), timeout=600)
        return r.returncode
    except subprocess.TimeoutExpired:
        log.error(f"{name} timed out after 600s")
        return 124
    except OSError as e:
        log.error(f"{name} failed to start: {e}")
        return 1


def master_tick():
    """Runs at :00 of every 5-hour bucket."""
    log.info("=== MASTER TICK ===")
    rc1 = run_script("gen-plan.py")
    if rc1 != 0:
        log.error(f"gen-plan exited {rc1}, skipping run-next-task")
        return
    rc2 = run_script("run-next-task.py")
    log.info(f"master tick done: gen-plan={rc1} run-next-task={rc2}")


def sub_tick():
    """Runs every 10 minutes."""
    log.info("=== SUB TICK ===")
    rc = run_script("run-next-task.py")
    log.info(f"sub tick done: rc={rc}")


def compute_next_wakeup(now):
    """Pick the next wakeup time (master or sub tick)."""
    bucket = current_bucket(now)
    # Master ticks: :00 of bucket hours (0, 5, 10, 15, 20)
    for offset in (0, 5, 10, 15, 20):
        master_at = now.replace(hour=offset, minute=0, second=0, microsecond=0)
        if master_at > now:
            return master_at, "master"
    # All master times today have passed → tomorrow's 00:00
    tomorrow = (now + timedelta(days=1)).replace(hour=0, minute=0, second=0, microsecond=0)
    return tomorrow, "master"


def main():
    log.info(f"=== 1Panel.edu scheduler daemon started ===")
    log.info(f"  pid: {__import__('os').getpid()}")
    log.info(f"  repo: {SCHEDULER_DIR.parent}")
    log.info(f"  schedule: master @ 0/5/10/15/20 :00 + sub @ every :00/:10/:20/:30/:40/:50")

    # Bootstrap: generate initial plan if state empty
    if not (SCHEDULER_DIR / "state.json").exists():
        log.info("no state.json, bootstrapping with gen-plan")
        run_script("gen-plan.py")
        # Run first task immediately
        run_script("run-next-task.py")

    while True:
        now = now_local()
        minute = now.minute
        hour = now.hour

        # Master tick at :00 of bucket hours
        if minute == 0 and hour % 5 == 0:
            master_tick()
            # Sleep to next :10
            next_wake = (now.replace(minute=10, second=0, microsecond=0)
                         if minute < 50
                         else (now + timedelta(hours=1)).replace(minute=0, second=0, microsecond=0))
            sleep_sec = seconds_until(next_wake)
            log.info(f"next wake at {next_wake.strftime('%H:%M:%S')} ({sleep_sec:.0f}s)")
            time.sleep(sleep_sec)
            continue

        # Sub tick at :00, :10, :20, :30, :40, :50
        if minute % 10 == 0:
            sub_tick()
            # Sleep to next :10
            next_minute = ((minute // 10) + 1) * 10
            if next_minute >= 60:
                next_wake = (now + timedelta(hours=1)).replace(minute=0, second=0, microsecond=0)
            else:
                next_wake = now.replace(minute=next_minute, second=0, microsecond=0)
            sleep_sec = seconds_until(next_wake)
            log.info(f"next wake at {next_wake.strftime('%H:%M:%S')} ({sleep_sec:.0f}s)")
            time.sleep(sleep_sec)
            continue

        # Not on a tick — sleep to next :00 or :10/:20/:30/:40/:50
        next_minute_options = []
        # next hour :00
        next_hour = (now + timedelta(hours=1)).replace(minute=0, second=0, microsecond=0)
        next_minute_options.append((next_hour, "next-hour-master-or-sub"))
        # next :10/:20/:30/:40/:50 within current hour
        for m in (10, 20, 30, 40, 50):
            if m > minute:
                w = now.replace(minute=m, second=0, microsecond=0)
                next_minute_options.append((w, "sub"))
                break
        next_wake, kind = min(next_minute_options, key=lambda x: x[0])
        sleep_sec = seconds_until(next_wake)
        log.info(f"next wake at {next_wake.strftime('%H:%M:%S')} ({sleep_sec:.0f}s, {kind})")
        time.sleep(sleep_sec)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        log.info("scheduler daemon stopped by user")
        sys.exit(0)
    except Exception as exc:
        log.exception(f"scheduler daemon crashed: {exc}")
        sys.exit(1)
