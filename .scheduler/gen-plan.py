#!/usr/bin/env python3
"""1Panel.edu scheduler: generate 5-hour window plan (v2).

9 sub-tasks, 30min apart. 5h window = 270min tasks + 30min buffer.
Schema version 2: adds circuit_breaker.

Pure code, no LLM. Designed for cron.
"""
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
STATE_FILE = SCHEDULER_DIR / "state.json"
HISTORY_MAX = 50  # 50 windows ~= 10 days at 5h window

# (task_id, name, offset_min from window_start)
# 9 tasks, 30min apart, 240min total task span, 60min buffer
SUB_TASKS = [
    ("index-rebuild",    "重建 KB 索引",                  0),
    ("quality-check",    "质量检查 (placeholder+断链)",   30),
    ("upstream-poll",    "1Panel 上游 commit 拉取 diff",  60),
    ("module-coverage",  "KB 模块 vs upstream 覆盖度",    90),
    ("health-score",     "KB 健康分 (0-100)",            120),
    ("stats-report",     "统计报告 (大小/文件)",         150),
    ("git-sync",         "GitHub 同步 (纯 Python)",      180),
    ("backup-snapshot",  "KB 快照 zip 备份",             210),
    ("daily-summary",    "当日 5 窗口汇总 (周日扩为周报)", 240),
]

SCHEMA_VERSION = 2
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
        return json.loads(STATE_FILE.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"warn: state.json unreadable ({exc}); rebuilding", file=sys.stderr)
        return None


def save_state(state):
    STATE_FILE.write_text(
        json.dumps(state, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def migrate_state(state):
    """Migrate state from v1 (no circuit_breaker) to v2."""
    if not state:
        return state
    if state.get("schema_version") == SCHEMA_VERSION:
        return state
    # v1 → v2: add circuit_breaker
    if "circuit_breaker" not in state:
        state["circuit_breaker"] = {}
    state["schema_version"] = SCHEMA_VERSION
    print(f"migrated state to schema v{SCHEMA_VERSION}", file=sys.stderr)
    return state


def append_history(state, prev_window):
    history = list(state.get("history", []))
    if prev_window:
        done = sum(1 for t in prev_window.get("sub_tasks", []) if t.get("status") == "done")
        failed = sum(1 for t in prev_window.get("sub_tasks", []) if t.get("status") == "failed")
        skipped = sum(1 for t in prev_window.get("sub_tasks", []) if t.get("status") == "skipped")
        history.append({
            "window_id": prev_window.get("window_id"),
            "completed_at": now_local().isoformat(),
            "tasks_done": done,
            "tasks_failed": failed,
            "tasks_skipped": skipped,
            "total": len(prev_window.get("sub_tasks", [])),
        })
    state["history"] = history[-HISTORY_MAX:]


def reset_circuit_breaker(state):
    """Reset circuit breaker at new window (per-window CB)."""
    state["circuit_breaker"] = {}


def main():
    now = now_local()
    state = load_state()
    if state:
        state = migrate_state(state)
    else:
        state = {"schema_version": SCHEMA_VERSION, "current_window": None, "history": [], "circuit_breaker": {}}

    # Append previous window to history before replacing
    append_history(state, state.get("current_window"))

    # Build new plan
    state["current_window"] = build_plan(now)

    # Reset circuit breaker on new window
    reset_circuit_breaker(state)

    state["schema_version"] = SCHEMA_VERSION
    save_state(state)

    plan = state["current_window"]
    print(f"plan generated: {plan['window_id']} ({len(SUB_TASKS)} sub-tasks)")
    print(f"  window: {plan['window_start'][:16]} -> {plan['window_end'][:16]}")
    for t in plan["sub_tasks"]:
        due = t["due_at"][11:16]  # HH:MM
        print(f"  [{due}] {t['id']}: {t['name']}")


if __name__ == "__main__":
    main()
