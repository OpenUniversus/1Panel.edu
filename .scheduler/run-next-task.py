#!/usr/bin/env python3
"""1Panel.edu scheduler: run next pending sub-task.
Pure code, no LLM. Designed for cron.
- Anti-re-run: only runs pending/failed tasks.
- Anti-miss: regenerates plan if window expired.
"""
import json
import os
import shutil
import subprocess
import sys
import time
import zipfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
STATE_FILE = SCHEDULER_DIR / "state.json"
REPO_ROOT = SCHEDULER_DIR.parent
TZ = timezone(timedelta(hours=8))


def now_local() -> datetime:
    return datetime.now(TZ)


def load_state():
    return json.loads(STATE_FILE.read_text(encoding="utf-8-sig"))


def save_state(state):
    STATE_FILE.write_text(
        json.dumps(state, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def window_expired(state) -> bool:
    cur = state.get("current_window")
    if not cur:
        return True
    return now_local() >= datetime.fromisoformat(cur["window_end"])


def find_next_task(state):
    """Pick first pending or failed sub-task (retryable)."""
    for t in state["current_window"]["sub_tasks"]:
        if t["status"] in ("pending", "failed"):
            return t
    return None


def mark_running(state, task_id):
    now = now_local().isoformat()
    for t in state["current_window"]["sub_tasks"]:
        if t["id"] == task_id:
            t["status"] = "running"
            t["started_at"] = now
            t["finished_at"] = None
            t["duration_sec"] = None
            t["result"] = None
            return


# ---------------------------------------------------------------------------
# Task implementations (each returns (ok: bool, msg: str))
# ---------------------------------------------------------------------------

def task_index_rebuild() -> tuple[bool, str]:
    """Rebuild KB-INDEX.md from current modules/ directory."""
    modules_dir = REPO_ROOT / "modules"
    total_md = total_html = total_size = 0
    lines = [
        "# KB 索引",
        "",
        f"生成时间: {now_local().isoformat()}",
        "",
        "| 模块 | HR | VA | 大小 |",
        "|---|---|---|---|",
    ]
    for m in sorted(p for p in modules_dir.iterdir() if p.is_dir()):
        hr = m / "HUMAN-READABLE.md"
        va = m / "visual-atlas.html"
        hr_ok = hr.exists()
        va_ok = va.exists()
        if hr_ok:
            total_md += 1
            total_size += hr.stat().st_size
        if va_ok:
            total_html += 1
            total_size += va.stat().st_size
        hr_mark = "OK" if hr_ok else "MISSING"
        va_mark = "OK" if va_ok else "MISSING"
        kb = ((hr.stat().st_size if hr_ok else 0) + (va.stat().st_size if va_ok else 0)) / 1024
        lines.append(f"| {m.name} | {hr_mark} | {va_mark} | {kb:.1f} KB |")
    lines.extend([
        "",
        f"**汇总**: {total_md} HR + {total_html} VA = {total_size / (1024*1024):.2f} MB",
    ])
    (REPO_ROOT / "KB-INDEX.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    return True, f"{total_md} modules, {total_size / (1024*1024):.2f} MB"


def task_quality_check() -> tuple[bool, str]:
    """Scan for typos/placeholders/missing files."""
    issues = []
    modules_dir = REPO_ROOT / "modules"
    for m in sorted(p for p in modules_dir.iterdir() if p.is_dir()):
        hr = m / "HUMAN-READABLE.md"
        va = m / "visual-atlas.html"
        if not hr.exists():
            issues.append(f"{m.name}: missing HUMAN-READABLE.md")
            continue
        if not va.exists():
            issues.append(f"{m.name}: missing visual-atlas.html")
        try:
            text = hr.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        for tag in ("TODO", "TBD", "FIXME", "XXX"):
            if tag in text:
                issues.append(f"{m.name}: has {tag}")
                break
    lines = [
        "# Quality Check",
        "",
        now_local().isoformat(),
        "",
    ]
    if issues:
        lines.extend(f"- {x}" for x in issues)
    else:
        lines.append("- OK: all modules complete, no placeholders found")
    (REPO_ROOT / "QUALITY-REPORT.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    if issues:
        return True, f"{len(issues)} issue(s) found"
    return True, "OK: clean"


def task_stats_report() -> tuple[bool, str]:
    """Generate STATS.md."""
    file_count = 0
    total_size = 0
    for p in REPO_ROOT.rglob("*"):
        if not p.is_file():
            continue
        if any(part in {".git", ".scheduler", ".backups"} for part in p.parts):
            continue
        if p.name == "daily-mgmt-state.json":
            continue
        file_count += 1
        total_size += p.stat().st_size
    module_count = sum(1 for p in (REPO_ROOT / "modules").iterdir() if p.is_dir())
    last_commit = ""
    try:
        out = subprocess.run(
            ["git", "-C", str(REPO_ROOT), "log", "-1", "--oneline"],
            capture_output=True, text=True, timeout=10, check=False,
        )
        last_commit = out.stdout.strip()
    except (OSError, subprocess.TimeoutExpired):
        pass
    text = (
        "# KB Stats\n\n"
        f"Generated: {now_local().isoformat()}\n\n"
        f"- Total modules: {module_count}\n"
        f"- Total files: {file_count}\n"
        f"- Repo size: {total_size / (1024*1024):.2f} MB\n"
        f"- Last commit: {last_commit}\n"
        "- GitHub: https://github.com/OpenUniversus/1Panel.edu\n"
    )
    (REPO_ROOT / "STATS.md").write_text(text, encoding="utf-8")
    return True, f"{file_count} files, {total_size / (1024*1024):.2f} MB"


def task_git_sync() -> tuple[bool, str]:
    """Run daily-mgmt.ps1 to push changes to GitHub."""
    bat = REPO_ROOT / "run-add-commit.bat"
    if not bat.exists():
        return False, "run-add-commit.bat not found"
    try:
        result = subprocess.run(
            ["cmd.exe", "/c", str(bat), "chore-daily-mgmt"],
            capture_output=True, text=True, timeout=120, cwd=str(REPO_ROOT),
        )
    except subprocess.TimeoutExpired:
        return False, "git-sync timeout"
    # Verify push: check if remote HEAD matches local HEAD
    try:
        local = subprocess.run(
            ["git", "-C", str(REPO_ROOT), "rev-parse", "HEAD"],
            capture_output=True, text=True, timeout=10, check=False,
        ).stdout.strip()
        remote = subprocess.run(
            ["git", "-C", str(REPO_ROOT), "rev-parse", "origin/main"],
            capture_output=True, text=True, timeout=10, check=False,
        ).stdout.strip()
    except OSError:
        return False, "git rev-parse failed"
    if local and remote and local == remote:
        return True, "git-sync OK (local == remote)"
    return True, f"committed locally; push status unclear (local={local[:7]} remote={remote[:7]})"


def task_backup_snapshot() -> tuple[bool, str]:
    """Create zip snapshot of modules/, keep last 10."""
    backups = REPO_ROOT / ".backups"
    backups.mkdir(exist_ok=True)
    stamp = now_local().strftime("%Y%m%d-%H%M")
    zip_path = backups / f"kb-snapshot-{stamp}.zip"
    if zip_path.exists():
        return True, f"already exists: {zip_path.name}"
    modules = REPO_ROOT / "modules"
    with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for p in modules.rglob("*"):
            if p.is_file():
                zf.write(p, p.relative_to(REPO_ROOT))
    # Prune old
    snaps = sorted(backups.glob("kb-snapshot-*.zip"), key=lambda x: x.stat().st_mtime, reverse=True)
    for old in snaps[10:]:
        old.unlink()
    return True, str(zip_path.relative_to(REPO_ROOT))


def task_audit_log(state) -> tuple[bool, str]:
    """Write WINDOW-AUDIT.md summarizing this window."""
    win = state["current_window"]
    done = sum(1 for t in win["sub_tasks"] if t["status"] == "done")
    total = len(win["sub_tasks"])
    lines = [
        "# Window Audit",
        "",
        f"Window: {win['window_id']}",
        f"{done} / {total} tasks done in this window",
        "",
        "## Status",
        "",
    ]
    for t in win["sub_tasks"]:
        dur = t.get("duration_sec")
        dur_s = f" ({dur}s)" if isinstance(dur, int) else ""
        lines.append(f"- [{t['status']:>8}] {t['id']}: {t['name']}{dur_s}")
    (REPO_ROOT / "WINDOW-AUDIT.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    return True, f"{done}/{total} done"


TASKS = {
    "index-rebuild":   task_index_rebuild,
    "quality-check":   task_quality_check,
    "stats-report":    task_stats_report,
    "git-sync":        task_git_sync,
    "backup-snapshot": task_backup_snapshot,
    "audit-log":       task_audit_log,
}


def run_task(state, task):
    task_id = task["id"]
    handler = TASKS.get(task_id)
    if handler is None:
        return False, f"no handler for {task_id}"
    try:
        # audit-log needs the state dict; others don't care
        if task_id == "audit-log":
            ok, msg = handler(state)
        else:
            ok, msg = handler()
    except Exception as exc:  # noqa: BLE001
        return False, f"{type(exc).__name__}: {exc}"
    return ok, msg


def finalize(state, task_id, ok, msg, duration_sec):
    now = now_local().isoformat()
    for t in state["current_window"]["sub_tasks"]:
        if t["id"] == task_id:
            t["status"] = "done" if ok else "failed"
            t["finished_at"] = now
            t["duration_sec"] = duration_sec
            t["result"] = msg
            return


def main():
    if not STATE_FILE.exists():
        # No plan yet — bootstrap
        subprocess.run(
            [sys.executable, str(SCHEDULER_DIR / "gen-plan.py")],
            check=True, timeout=30,
        )
    state = load_state()

    # Anti-miss: window expired → regenerate
    if window_expired(state):
        print(f"[{now_local().strftime('%H:%M')}] window expired, regenerating plan")
        subprocess.run(
            [sys.executable, str(SCHEDULER_DIR / "gen-plan.py")],
            check=True, timeout=30,
        )
        state = load_state()

    # Anti-re-run: only pending/failed
    next_task = find_next_task(state)
    if next_task is None:
        total = len(state["current_window"]["sub_tasks"])
        print(f"<mavis-progress>silent: window {state['current_window']['window_id']} all done ({total}/{total})</mavis-progress>")
        return 0

    task_id = next_task["id"]
    mark_running(state, task_id)
    save_state(state)

    # Run task
    start_time = time.time()
    ts = now_local().strftime("%H:%M:%S")
    print(f"[{ts}] running: {task_id} ({next_task['name']})")
    ok, msg = run_task(state, next_task)
    duration = int(time.time() - start_time)
    finalize(state, task_id, ok, msg, duration)
    save_state(state)

    ts = now_local().strftime("%H:%M:%S")
    status = "done" if ok else "failed"
    print(f"[{ts}] {status}: {task_id} ({msg}) [{duration}s]")
    if ok:
        print(f"<mavis-progress>tick: {task_id} {msg} [{duration}s]</mavis-progress>")
        return 0
    print(f"<mavis-progress>tick: {task_id} FAILED: {msg}</mavis-progress>")
    return 1


if __name__ == "__main__":
    sys.exit(main())
