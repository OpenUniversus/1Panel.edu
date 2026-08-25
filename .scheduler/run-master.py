#!/usr/bin/env python3
"""1Panel.edu scheduler: master tick runner.

Chains gen-plan + run-next-task. Used by:
- Manual invocation (python run-master.py)
- scheduler-daemon.py master_tick() (which inlines the same logic)

Pure code, no LLM. Designed for cron / manual.
"""
import subprocess
import sys
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
GEN_PLAN = SCHEDULER_DIR / "gen-plan.py"
RUN_NEXT = SCHEDULER_DIR / "run-next-task.py"


def main():
    if not GEN_PLAN.exists() or not RUN_NEXT.exists():
        print("error: gen-plan.py or run-next-task.py missing", file=sys.stderr)
        return 2

    # Step 1: gen-plan
    print("[run-master] step 1/2: gen-plan.py")
    r1 = subprocess.run([sys.executable, str(GEN_PLAN)], cwd=str(SCHEDULER_DIR), timeout=30)
    if r1.returncode != 0:
        print(f"[run-master] gen-plan.py failed (rc={r1.returncode}), aborting", file=sys.stderr)
        return r1.returncode

    # Step 2: run-next-task
    print("[run-master] step 2/2: run-next-task.py")
    r2 = subprocess.run([sys.executable, str(RUN_NEXT)], cwd=str(SCHEDULER_DIR), timeout=600)
    if r2.returncode != 0:
        print(f"[run-master] run-next-task.py failed (rc={r2.returncode})", file=sys.stderr)
        return r2.returncode

    print("[run-master] done")
    return 0


if __name__ == "__main__":
    sys.exit(main())
