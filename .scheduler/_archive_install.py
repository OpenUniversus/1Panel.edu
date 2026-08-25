#!/usr/bin/env python3
"""Install Windows Task Scheduler entries for 1Panel.edu scheduler.
Pure Python. Uses subprocess to call schtasks.exe. Idempotent.
Run: python install.py
"""
import shutil
import subprocess
import sys
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
REPO_ROOT     = SCHEDULER_DIR.parent
PYTHON_BIN    = shutil.which("python") or "C:\\Python312\\python.exe"
# Resolve to canonical Windows path
PY_WIN        = str(Path(PYTHON_BIN).resolve()) if Path(PYTHON_BIN).exists() else PYTHON_BIN

# Tasks: (name, schedule, modifier, command)
TASKS = [
    {
        "name":     "master-plan",
        "schedule": "HOURLY",
        "modifier": "5",          # every 5 hours
        "start":    "00:00",
        "command":  f'"{PY_WIN}" "{SCHEDULER_DIR}\\run-master.py"',
    },
    {
        "name":     "sub-task",
        "schedule": "MINUTE",
        "modifier": "10",         # every 10 minutes
        "start":    "00:00",
        "command":  f'"{PY_WIN}" "{SCHEDULER_DIR}\\run-next-task.py"',
    },
]


def log(msg):
    print(f"[install] {msg}", flush=True)


def task_exists(name):
    r = subprocess.run(
        ["schtasks", "/Query", "/TN", f"\\1Panel-Edu\\{name}"],
        capture_output=True, text=True,
    )
    return r.returncode == 0


def delete_task(name):
    if task_exists(name):
        subprocess.run(
            ["schtasks", "/Delete", "/TN", f"\\1Panel-Edu\\{name}", "/F"],
            capture_output=True, text=True,
        )
        log(f"removed existing: \\1Panel-Edu\\{name}")


def create_task(spec):
    name = spec["name"]
    delete_task(name)

    cmd = [
        "schtasks", "/Create",
        "/TN", f"\\1Panel-Edu\\{name}",
        "/TR", spec["command"],
        "/SC", spec["schedule"],
        "/MO", spec["modifier"],
        "/ST", spec["start"],
        "/F",   # force overwrite
        "/RL", "HIGHEST",
    ]
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        log(f"FAILED: {name}: {r.stderr.strip() or r.stdout.strip()}")
        return False
    log(f"registered: \\1Panel-Edu\\{name} (every {spec['modifier']} {spec['schedule']} from {spec['start']})")
    return True


def main():
    log(f"python: {PY_WIN}")
    log(f"repo:   {REPO_ROOT}")
    log(f"log:    {SCHEDULER_DIR / 'logs'}")
    (SCHEDULER_DIR / "logs").mkdir(exist_ok=True)

    ok = True
    for spec in TASKS:
        if not create_task(spec):
            ok = False

    print()
    if ok:
        print("=== 1Panel.edu cron installed (100% code, no LLM) ===")
        print("  master-plan:  every 5 hours at :00 (0, 5, 10, 15, 20)")
        print("  sub-task:     every 10 minutes")
        print()
        print("Verify: schtasks /Query /TN '\\1Panel-Edu\\*'")
    else:
        print("=== install completed with errors ===")
        sys.exit(1)


if __name__ == "__main__":
    main()
