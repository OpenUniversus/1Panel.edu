#!/usr/bin/env python3
"""1Panel.edu scheduler: generate 5-hour window plan.
Pure code, no LLM. Designed for cron.
"""
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
STATE_FILE = SCHEDULER_DIR / "state.json"
HISTORY_MAX = 20

SUB_TASKS = [
    ("index-rebuild",   "重建 KB 索引",         0),
    ("quality-check",   "质量检查 (typo/断链)", 50),
    ("stats-report",    "统计报告 (大小/文件)", 100),
    ("git-sync",        "GitHub 同步 (daily-mgmt)", 150),
    ("backup-snapshot", "KB 快照备份",         200),
    ("audit-log",       "窗口审计日志",         250),
]

# Asia/Shanghai (UTC+8) is the project default.
TZ = timezone(timedelta(hours=8))


def now_local() -> datetime:
    return datetime.now(TZ)


def current_window_bounds(now: datetime):
    """5-hour bucket: 0-4 / 5-9 / 10-14 / 15-19 / 20-23."""
    bucket_hour = (now.hour // 5) * 5
    start = now.replace(hour=bucket_hour, minute=0, second=0, microsecond=0)
    end = start + timedelta(hours=5)
    return start, end


def build_plan(now: datetime):
    start, end = current_window_bounds(now)
    sub_tasks = []
    for task_id, name, offset_min in SUB_TASKS:
        due_at = start + timedelta(minutes=offset_min)
        sub_tasks.append({
            "id": task_id,
            "name": name,
            "due_at": due_at.isoformat(),
            "status": "pending",
            "started_at": None,
            "finished_at": None,
            "duration_sec": None,
            "result": None,
        })
    return {
        "window_id": f"win-{start.strftime('%Y%m%d-%H%M')}",
        "window_start": start.isoformat(),
        "window_end": end.isoformat(),
        "generated_at": now.isoformat(),
        "sub_tasks": sub_tasks,
    }


def load_state():
    if not STATE_FILE.exists():
        return None
    try:
        # utf-8-sig strips BOM if present (compat with prior PowerShell writes)
        return json.loads(STATE_FILE.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"warn: state.json unreadable ({exc}); rebuilding", file=sys.stderr)
        return None


def save_state(state):
    STATE_FILE.write_text(
        json.dumps(state, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def append_history(state, prev_window):
    history = list(state.get("history", []))
    if prev_window:
        done = sum(1 for t in prev_window.get("sub_tasks", []) if t.get("status") == "done")
        failed = sum(1 for t in prev_window.get("sub_tasks", []) if t.get("status") == "failed")
        history.append({
            "window_id": prev_window.get("window_id"),
            "completed_at": now_local().isoformat(),
            "tasks_done": done,
            "tasks_failed": failed,
        })
    state["history"] = history[-HISTORY_MAX:]


def main():
    now = now_local()
    state = load_state() or {"schema_version": 1, "current_window": None, "history": []}
    append_history(state, state.get("current_window"))
    state["current_window"] = build_plan(now)
    state["schema_version"] = 1
    save_state(state)

    plan = state["current_window"]
    print(f"plan generated: {plan['window_id']} ({len(SUB_TASKS)} sub-tasks)")
    print(f"  window: {plan['window_start'][:16]} -> {plan['window_end'][:16]}")
    for t in plan["sub_tasks"]:
        due = t["due_at"][11:16]  # HH:MM
        print(f"  [{due}] {t['id']}: {t['name']}")


if __name__ == "__main__":
    main()
