#!/usr/bin/env python3
"""1Panel.edu scheduler daemon (v2).

Pure Python. Single long-running process. No LLM, no admin, no schtasks.
- Master tick at :00 of every 5-hour bucket (0, 5, 10, 15, 20):
    runs gen-plan + run-next-task
    Also resets daily ALERTS.md at 00:00
- Sub tick at :00 (non-bucket), :15, :30, :45 of every hour:
    runs run-next-task
- Lock file anti-overlap (run-next-task.py owns)
- PID file: .scheduler/scheduler.pid (for monitor-daemon to track)
- Crash-safe: re-loads state on each tick
- Graceful shutdown on SIGINT/SIGTERM (best-effort on Windows)

Run: python scheduler-daemon.py
Stop: Ctrl-C (or kill <pid>)
"""
import atexit
import logging
import os
import signal
import subprocess
import sys
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
LOG_DIR = SCHEDULER_DIR / "logs"
LOG_DIR.mkdir(exist_ok=True)
SCHEDULER_PID_FILE = SCHEDULER_DIR / "scheduler.pid"

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
REPO_ROOT = SCHEDULER_DIR.parent
_running = True  # for graceful shutdown


def now_local():
    return datetime.now(TZ)


def current_bucket(now):
    """5-hour bucket: 0-4 / 5-9 / 10-14 / 15-19 / 20-23."""
    return (now.hour // 5) * 5


def is_master_tick(now):
    """Master = :00 of bucket hour (0, 5, 10, 15, 20)."""
    return now.minute == 0 and current_bucket(now) == now.hour


def is_sub_tick(now):
    """Sub = :15, :30, :45, plus :00 of non-bucket hours."""
    if now.minute == 0:
        return not is_master_tick(now)
    return now.minute in (15, 30, 45)


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


def reset_daily_alerts():
    """At master tick 00:00, clear ALERTS.md (new day)."""
    alerts = REPO_ROOT / "ALERTS.md"
    if not alerts.exists():
        return
    try:
        # Just truncate; keep header
        content = alerts.read_text(encoding="utf-8", errors="ignore")
        lines = content.splitlines()
        # Keep only header (first 3 lines)
        if lines and lines[0].startswith("# KB Alerts"):
            new_content = "\n".join(lines[:3]) + "\n\n"
        else:
            new_content = "# KB Alerts\n\nCumulative task failures. Resets daily at 00:00 Asia/Shanghai.\n\n"
        alerts.write_text(new_content, encoding="utf-8")
        log.info("daily ALERTS.md reset")
    except OSError as e:
        log.warning(f"could not reset ALERTS.md: {e}")


def master_tick():
    """Runs at :00 of every 5-hour bucket."""
    log.info("=== MASTER TICK ===")
    # Daily reset at 00:00 master tick
    if now_local().hour == 0:
        reset_daily_alerts()
    rc1 = run_script("gen-plan.py")
    if rc1 != 0:
        log.error(f"gen-plan exited {rc1}, skipping run-next-task")
        return
    rc2 = run_script("run-next-task.py")
    log.info(f"master tick done: gen-plan={rc1} run-next-task={rc2}")


def sub_tick():
    """Runs at :15, :30, :45 and :00 of non-bucket hours."""
    log.info("=== SUB TICK ===")
    rc = run_script("run-next-task.py")
    log.info(f"sub tick done: rc={rc}")


def next_wakeup(now):
    """Compute the next wakeup time and kind."""
    # Build list of candidates: next master and next sub
    candidates = []
    # Master tick: next bucket-hour :00
    next_master_hour = current_bucket(now) + 5
    if next_master_hour > 20:
        next_master_hour = 0
        next_master_day = now + timedelta(days=1)
    else:
        next_master_day = now
    master_at = next_master_day.replace(hour=next_master_hour, minute=0, second=0, microsecond=0)
    if master_at > now:
        candidates.append((master_at, "master"))
    # Sub tick: next :15, :30, :45, or :00 (non-master)
    for m in (15, 30, 45):
        if m > now.minute:
            w = now.replace(minute=m, second=0, microsecond=0)
            candidates.append((w, "sub"))
            break
    else:
        # No more :15/:30/:45 this hour → next hour :00 (or master)
        if (now + timedelta(hours=1)).hour == next_master_hour and now.minute >= 45:
            # Next hour is master; already added
            pass
        else:
            next_hour = (now + timedelta(hours=1)).replace(minute=0, second=0, microsecond=0)
            candidates.append((next_hour, "sub"))
    # If minute is in (0..14), next sub candidate is :15 within current hour
    # If minute in (15..29), next is :30
    # If minute in (30..44), next is :45
    # If minute in (45..59), next is next-hour :00 (or master)
    if now.minute < 15:
        w = now.replace(minute=15, second=0, microsecond=0)
        if w > now:
            candidates.append((w, "sub"))
    elif now.minute < 30:
        w = now.replace(minute=30, second=0, microsecond=0)
        candidates.append((w, "sub"))
    elif now.minute < 45:
        w = now.replace(minute=45, second=0, microsecond=0)
        candidates.append((w, "sub"))
    next_wake, kind = min(candidates, key=lambda x: x[0])
    return next_wake, kind


def _shutdown(signum, frame):
    global _running
    log.info(f"received signal {signum}, shutting down gracefully")
    _running = False


def _cleanup_pid():
    """Remove scheduler.pid on exit."""
    try:
        if SCHEDULER_PID_FILE.exists():
            SCHEDULER_PID_FILE.unlink()
            log.info("removed scheduler.pid")
    except OSError:
        pass


def main():
    # Try to register signal handlers (best-effort on Windows)
    try:
        signal.signal(signal.SIGINT, _shutdown)
        if hasattr(signal, "SIGTERM"):
            signal.signal(signal.SIGTERM, _shutdown)
    except (ValueError, OSError):
        pass

    log.info("=== 1Panel.edu scheduler daemon v2 started ===")
    log.info(f"  pid: {os.getpid()}")
    log.info(f"  repo: {REPO_ROOT}")
    log.info("  schedule: master @ 0/5/10/15/20 :00, sub @ :00(non-bucket)/:15/:30/:45")

    # Write PID file (for monitor-daemon)
    SCHEDULER_PID_FILE.write_text(str(os.getpid()), encoding="utf-8")
    atexit.register(_cleanup_pid)

    # Bootstrap
    if not (SCHEDULER_DIR / "state.json").exists():
        log.info("no state.json, bootstrapping with gen-plan")
        run_script("gen-plan.py")
        run_script("run-next-task.py")

    while _running:
        now = now_local()

        if is_master_tick(now):
            master_tick()
        elif is_sub_tick(now):
            sub_tick()

        # Sleep until next wakeup
        now = now_local()
        next_wake, kind = next_wakeup(now)
        sleep_sec = seconds_until(next_wake)
        log.info(f"next wake at {next_wake.strftime('%H:%M:%S')} ({sleep_sec:.0f}s, {kind})")
        # Sleep in small chunks so signal can interrupt
        slept = 0.0
        chunk = 5.0
        while slept < sleep_sec and _running:
            time.sleep(min(chunk, sleep_sec - slept))
            slept += chunk

    log.info("scheduler daemon stopped")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        log.info("scheduler daemon stopped by user (KeyboardInterrupt)")
        sys.exit(0)
    except Exception as exc:
        log.exception(f"scheduler daemon crashed: {exc}")
        sys.exit(1)
