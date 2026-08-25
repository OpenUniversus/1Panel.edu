#!/usr/bin/env python3
"""Master tick: gen-plan + run-next-task, chained.
Used by Windows Task Scheduler at 0/5/10/15/20 hours.
"""
import subprocess
import sys
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent


def main():
    plan = SCHEDULER_DIR / "gen-plan.py"
    runner = SCHEDULER_DIR / "run-next-task.py"
    for script in (plan, runner):
        r = subprocess.run(
            [sys.executable, str(script)],
            cwd=str(SCHEDULER_DIR),
        )
        if r.returncode != 0:
            print(f"master: {script.name} exited {r.returncode}", flush=True)
            return r.returncode
    return 0


if __name__ == "__main__":
    sys.exit(main())
