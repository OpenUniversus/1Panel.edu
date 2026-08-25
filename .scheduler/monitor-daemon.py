#!/usr/bin/env python3
"""1Panel.edu monitor daemon (watchdog).

Watches scheduler-daemon.py. If it's not running, starts it.
Single long-running process. No LLM, no admin, no schtasks.

- Reads .scheduler/scheduler.pid (written by scheduler-daemon.py)
- Every 60s: check if PID alive
- If dead: spawn new scheduler-daemon.py
- Cooldown 60s after restart (avoid restart storm)
- Own PID file: .scheduler/monitor.pid (prevents multi-instance)

Run: python monitor-daemon.py
Stop: Ctrl-C (or kill <pid>)
"""
import atexit
import ctypes
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
MONITOR_PID_FILE = SCHEDULER_DIR / "monitor.pid"
SCHEDULER_SCRIPT = SCHEDULER_DIR / "scheduler-daemon.py"
TZ = timezone(timedelta(hours=8))

CHECK_INTERVAL_SEC = 60
RESTART_COOLDOWN_SEC = 60

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.FileHandler(LOG_DIR / "monitor.log", encoding="utf-8"),
        logging.StreamHandler(sys.stdout),
    ],
)
log = logging.getLogger("monitor")
_running = True


def now_local():
    return datetime.now(TZ)


def _read_pid_file(path: Path) -> int:
    if not path.exists():
        return 0
    try:
        return int(path.read_text(encoding="utf-8-sig").strip())
    except (OSError, ValueError):
        return 0


def pid_alive(pid: int) -> bool:
    """Windows: OpenProcess query."""
    if pid <= 0:
        return False
    try:
        kernel32 = ctypes.windll.kernel32
        PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
        h = kernel32.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, False, pid)
        if h:
            kernel32.CloseHandle(h)
            return True
        return False
    except Exception:
        return False


def start_scheduler():
    """Spawn scheduler-daemon.py as detached subprocess."""
    log.info("starting scheduler-daemon.py...")
    try:
        # CREATE_NEW_PROCESS_GROUP = 0x00000200, DETACHED_PROCESS = 0x00000008
        # Use DETACHED so scheduler survives monitor exit (best-effort on Windows)
        DETACHED_PROCESS = 0x00000008
        CREATE_NEW_PROCESS_GROUP = 0x00000200
        subprocess.Popen(
            [sys.executable, str(SCHEDULER_SCRIPT)],
            cwd=str(SCHEDULER_DIR),
            creationflags=DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP,
            close_fds=True,
        )
        log.info("scheduler-daemon.py spawned")
    except OSError as e:
        log.error(f"failed to spawn scheduler: {e}")


def check_and_restart(last_restart_at: float) -> float:
    """Returns new last_restart_at."""
    pid = _read_pid_file(SCHEDULER_PID_FILE)
    if pid == 0:
        log.info("scheduler.pid missing")
    elif pid_alive(pid):
        log.debug(f"scheduler pid={pid} alive")
        return last_restart_at
    else:
        log.warning(f"scheduler pid={pid} dead")
    # Stale pid file
    try:
        if SCHEDULER_PID_FILE.exists():
            SCHEDULER_PID_FILE.unlink()
    except OSError:
        pass
    # Cooldown check
    now = time.time()
    if now - last_restart_at < RESTART_COOLDOWN_SEC:
        log.info(f"cooldown active ({RESTART_COOLDOWN_SEC - (now - last_restart_at):.0f}s remaining); skip restart")
        return last_restart_at
    start_scheduler()
    return now


def _cleanup_monitor_pid():
    try:
        if MONITOR_PID_FILE.exists() and _read_pid_file(MONITOR_PID_FILE) == os.getpid():
            MONITOR_PID_FILE.unlink()
            log.info("removed monitor.pid")
    except OSError:
        pass


def _shutdown(signum, frame):
    global _running
    log.info(f"received signal {signum}, shutting down")
    _running = False


def main():
    # Single-instance check
    existing = _read_pid_file(MONITOR_PID_FILE)
    if existing and pid_alive(existing) and existing != os.getpid():
        print(f"monitor-daemon already running at pid={existing}; abort", file=sys.stderr)
        sys.exit(1)

    # Signal handlers
    try:
        signal.signal(signal.SIGINT, _shutdown)
        if hasattr(signal, "SIGTERM"):
            signal.signal(signal.SIGTERM, _shutdown)
    except (ValueError, OSError):
        pass

    MONITOR_PID_FILE.write_text(str(os.getpid()), encoding="utf-8")
    atexit.register(_cleanup_monitor_pid)

    log.info("=== 1Panel.edu monitor daemon v2 started ===")
    log.info(f"  pid: {os.getpid()}")
    log.info(f"  scheduler dir: {SCHEDULER_DIR}")
    log.info(f"  check every {CHECK_INTERVAL_SEC}s, restart cooldown {RESTART_COOLDOWN_SEC}s")

    # If scheduler not running, start it
    last_restart_at = 0.0
    if not pid_alive(_read_pid_file(SCHEDULER_PID_FILE)):
        log.info("scheduler not running on startup, starting")
        start_scheduler()
        last_restart_at = time.time()

    while _running:
        last_restart_at = check_and_restart(last_restart_at)
        # Sleep in small chunks
        slept = 0.0
        chunk = 5.0
        while slept < CHECK_INTERVAL_SEC and _running:
            time.sleep(min(chunk, CHECK_INTERVAL_SEC - slept))
            slept += chunk

    log.info("monitor daemon stopped")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        log.info("monitor daemon stopped by user (KeyboardInterrupt)")
        sys.exit(0)
    except Exception as exc:
        log.exception(f"monitor daemon crashed: {exc}")
        sys.exit(1)
