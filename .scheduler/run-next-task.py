#!/usr/bin/env python3
"""1Panel.edu scheduler: run next pending sub-task (v2).

Features:
- 9 task handlers (4 new, 1 extended, 1 changed to pure Python, 3 unchanged)
- Lock file anti-overlap
- Circuit breaker (3 consecutive failures → excluded for window)
- Alert log (cumulative ALERTS.md, daily reset at master tick 00:00)
- Anti-re-run: only pending/failed (skips excluded)
- Anti-miss: window expired → regenerate via gen-plan.py

Pure code, no LLM. Designed for cron.
"""
import json
import os
import subprocess
import sys
import time
import urllib.request
import zipfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

SCHEDULER_DIR = Path(__file__).parent
STATE_FILE    = SCHEDULER_DIR / "state.json"
LOCK_FILE     = SCHEDULER_DIR / "lock"
LOG_DIR       = SCHEDULER_DIR / "logs"
LOG_DIR.mkdir(exist_ok=True)
REPO_ROOT     = SCHEDULER_DIR.parent
TZ            = timezone(timedelta(hours=8))
LOCK_MAX_AGE_SEC = 30 * 60  # 30 min, then force-kill
UPSTREAM_REPO  = "https://github.com/1Panel-dev/1Panel.git"
UPSTREAM_BRANCH = "dev-v2"
UPSTREAM_STATE_FILE = SCHEDULER_DIR / ".upstream-state.json"


# ---------------------------------------------------------------------------
# Time + state helpers
# ---------------------------------------------------------------------------

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
    """Pick first pending or failed sub-task, skipping excluded (CB)."""
    cb = state.get("circuit_breaker", {})
    for t in state["current_window"]["sub_tasks"]:
        if t["status"] in ("pending", "failed"):
            cb_entry = cb.get(t["id"], {})
            if cb_entry.get("excluded"):
                continue
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
# Lock file
# ---------------------------------------------------------------------------

def _read_lock():
    if not LOCK_FILE.exists():
        return None
    try:
        return json.loads(LOCK_FILE.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError):
        return None


def _pid_alive(pid: int) -> bool:
    if pid <= 0:
        return False
    try:
        # Windows: OpenProcess check
        import ctypes
        kernel32 = ctypes.windll.kernel32
        PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
        h = kernel32.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, False, pid)
        if h:
            kernel32.CloseHandle(h)
            return True
        return False
    except Exception:
        return False


def acquire_lock(task_id: str) -> bool:
    """Return True if lock acquired, False if blocked by stale lock."""
    existing = _read_lock()
    if existing:
        pid = existing.get("pid", 0)
        started = existing.get("started_at")
        try:
            age = (now_local() - datetime.fromisoformat(started)).total_seconds()
        except Exception:
            age = LOCK_MAX_AGE_SEC + 1
        if _pid_alive(pid) and age < LOCK_MAX_AGE_SEC:
            print(f"[{now_local().strftime('%H:%M')}] lock held by pid={pid} age={age:.0f}s task={existing.get('task_id')}; skip")
            return False
        # Stale or dead: nuke it
        print(f"[{now_local().strftime('%H:%M')}] stale lock (pid={pid} alive={_pid_alive(pid)} age={age:.0f}s); removing")
        if _pid_alive(pid):
            try:
                import ctypes
                ctypes.windll.kernel32.TerminateProcess(pid, 1)
            except Exception:
                pass
        try:
            LOCK_FILE.unlink()
        except OSError:
            pass
    LOCK_FILE.write_text(
        json.dumps({
            "pid": os.getpid(),
            "started_at": now_local().isoformat(),
            "task_id": task_id,
        }, ensure_ascii=False),
        encoding="utf-8",
    )
    return True


def release_lock():
    try:
        LOCK_FILE.unlink()
    except OSError:
        pass


# ---------------------------------------------------------------------------
# Circuit breaker + alert log
# ---------------------------------------------------------------------------

CB_FAIL_THRESHOLD = 3


def on_task_complete(state, task_id: str, ok: bool, msg: str):
    """Update circuit breaker state after a task completes."""
    cb = state.setdefault("circuit_breaker", {})
    entry = cb.setdefault(task_id, {"consecutive_failures": 0, "first_failure": None, "excluded": False})
    if ok:
        entry["consecutive_failures"] = 0
        entry["excluded"] = False
        return
    entry["consecutive_failures"] += 1
    if entry["first_failure"] is None:
        entry["first_failure"] = now_local().isoformat()
    if entry["consecutive_failures"] >= CB_FAIL_THRESHOLD and not entry["excluded"]:
        entry["excluded"] = True
        _write_blacklist(state)
        print(f"[{now_local().strftime('%H:%M')}] CB: {task_id} excluded (3 consecutive failures)")


def _write_blacklist(state):
    cb = state.get("circuit_breaker", {})
    excluded = [tid for tid, e in cb.items() if e.get("excluded")]
    if not excluded:
        return
    lines = [
        "# Task Blacklist (Circuit Breaker)",
        "",
        f"Generated: {now_local().isoformat()}",
        "",
        "Tasks excluded from this window due to 3+ consecutive failures:",
        "",
    ]
    for tid in excluded:
        entry = cb.get(tid, {})
        lines.append(f"- **{tid}**: {entry.get('consecutive_failures')} failures, first={entry.get('first_failure')}")
    lines.append("")
    lines.append("Will reset on next 5h window (master tick).")
    (REPO_ROOT / "BLACKLIST.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def append_alert(task_id: str, msg: str):
    """Append to ALERTS.md (cumulative, daily reset at master tick 00:00)."""
    alerts_path = REPO_ROOT / "ALERTS.md"
    if not alerts_path.exists():
        alerts_path.write_text(
            "# KB Alerts\n\nCumulative task failures. Resets daily at 00:00 Asia/Shanghai.\n\n",
            encoding="utf-8",
        )
    with alerts_path.open("a", encoding="utf-8") as f:
        f.write(f"- [{now_local().isoformat()}] **{task_id}** FAILED: {msg}\n")


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


def _scan_markdown_links(text: str, base_dir: Path) -> list[tuple[str, str]]:
    """Return list of (line_no, link) for relative markdown links that don't resolve."""
    import re
    md_link = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")
    issues = []
    for i, line in enumerate(text.splitlines(), 1):
        for m in md_link.finditer(line):
            target = m.group(2).strip()
            # Skip URLs (http/https/mailto/#anchor)
            if "://" in target or target.startswith(("mailto:", "#")):
                continue
            # Strip anchor
            target_path = target.split("#", 1)[0]
            if not target_path:
                continue
            resolved = (base_dir / target_path).resolve()
            if not resolved.exists():
                issues.append((str(i), target))
    return issues


def _scan_html_refs(html_path: Path) -> list[tuple[str, str]]:
    """Return list of (tag, ref) for HTML src/href that don't resolve (local only)."""
    import re
    try:
        text = html_path.read_text(encoding="utf-8", errors="ignore")
    except OSError:
        return []
    issues = []
    # Match src="..." and href="..."
    for tag, attr in (("img", "src"), ("a", "href"), ("script", "src"), ("link", "href")):
        pattern = re.compile(rf'<{tag}\b[^>]*\b{attr}=["\']([^"\']+)["\']', re.IGNORECASE)
        for m in pattern.finditer(text):
            ref = m.group(1).strip()
            if "://" in ref or ref.startswith(("data:", "#", "mailto:")):
                continue
            ref_clean = ref.split("#", 1)[0]
            if not ref_clean:
                continue
            resolved = (html_path.parent / ref_clean).resolve()
            if not resolved.exists():
                issues.append((tag, ref))
    return issues


def _check_dead_links(text: str, timeout: float = 3.0) -> list[tuple[str, str]]:
    """Return list of (line_no, url) for HTTP links that fail HEAD/GET."""
    import re
    url_pat = re.compile(r"https?://[^\s\)\]\"'<>]+")
    urls = set()
    for m in url_pat.finditer(text):
        urls.add(m.group(0).rstrip(".,;:"))
    dead = []
    for url in sorted(urls):
        try:
            req = urllib.request.Request(url, method="HEAD")
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                if resp.status >= 400:
                    dead.append(("?", url))
        except Exception:
            dead.append(("?", url))
    return dead


def task_quality_check() -> tuple[bool, str]:
    """Scan for placeholders + markdown broken links + HTML broken refs + dead external links."""
    issues: list[str] = []
    modules_dir = REPO_ROOT / "modules"
    placeholder_tags = ("TODO", "TBD", "FIXME", "XXX", "HACK")
    external_check_done = False
    for m in sorted(p for p in modules_dir.iterdir() if p.is_dir()):
        hr = m / "HUMAN-READABLE.md"
        va = m / "visual-atlas.html"
        if not hr.exists():
            issues.append(f"{m.name}: missing HUMAN-READABLE.md")
            continue
        if not va.exists():
            issues.append(f"{m.name}: missing visual-atlas.html")
        try:
            hr_text = hr.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        # 1. Placeholder scan
        for tag in placeholder_tags:
            if tag in hr_text:
                issues.append(f"{m.name}: has placeholder '{tag}'")
                break
        # 2. Markdown broken links
        broken = _scan_markdown_links(hr_text, hr.parent)
        for line_no, link in broken:
            issues.append(f"{m.name} HR.md L{line_no}: broken link '{link}'")
        # 3. HTML broken refs
        if va.exists():
            html_broken = _scan_html_refs(va)
            for tag, ref in html_broken:
                issues.append(f"{m.name} visual-atlas.html: <{tag} {ref}> missing")
    # 4. Dead external links (only once, on README.md)
    if not external_check_done:
        readme = REPO_ROOT / "README.md"
        if readme.exists():
            try:
                readme_text = readme.read_text(encoding="utf-8", errors="ignore")
                dead = _check_dead_links(readme_text)
                for _, url in dead:
                    issues.append(f"README.md: dead external link '{url}'")
            except OSError:
                pass
        external_check_done = True

    lines = [
        "# Quality Check",
        "",
        now_local().isoformat(),
        "",
    ]
    if issues:
        lines.extend(f"- {x}" for x in issues)
    else:
        lines.append("- OK: all modules complete, no placeholders/broken links found")
    (REPO_ROOT / "QUALITY-REPORT.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    if issues:
        return True, f"{len(issues)} issue(s) found"
    return True, "OK: clean"


def _read_upstream_state() -> dict:
    if not UPSTREAM_STATE_FILE.exists():
        return {}
    try:
        return json.loads(UPSTREAM_STATE_FILE.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError):
        return {}


def _write_upstream_state(state: dict):
    UPSTREAM_STATE_FILE.write_text(
        json.dumps(state, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def task_upstream_poll() -> tuple[bool, str]:
    """git ls-remote upstream dev-v2, write UPSTREAM-DIFF.md."""
    try:
        result = subprocess.run(
            ["git", "ls-remote", UPSTREAM_REPO, f"refs/heads/{UPSTREAM_BRANCH}"],
            capture_output=True, text=True, timeout=15, check=False,
        )
    except subprocess.TimeoutExpired:
        return True, "upstream ls-remote timeout (network slow); skipping"
    except OSError as e:
        return True, f"git ls-remote failed to start: {e}"
    if result.returncode != 0 or not result.stdout.strip():
        return True, f"upstream unreachable (rc={result.returncode}); skipping"
    remote_sha = result.stdout.strip().split()[0]
    prev = _read_upstream_state()
    prev_sha = prev.get("sha")
    prev_checked = prev.get("checked_at")
    try:
        last_seen_dt = datetime.fromisoformat(prev_checked) if prev_checked else None
    except Exception:
        last_seen_dt = None
    now = now_local()
    stale_days = (now - last_seen_dt).days if last_seen_dt else None
    lines = [
        "# Upstream Diff",
        "",
        f"Checked: {now.isoformat()}",
        f"Upstream: `{UPSTREAM_REPO}` branch `{UPSTREAM_BRANCH}`",
        f"Latest commit: `{remote_sha[:12]}`",
        "",
    ]
    if prev_sha == remote_sha:
        lines.append(f"No change since last check ({prev_checked}).")
        if stale_days is not None and stale_days >= 7:
            lines.append(f"**Stale: {stale_days} days since upstream activity** (consider syncing KB)")
    else:
        if prev_sha:
            lines.append(f"**Upstream changed!** `{prev_sha[:12]}` -> `{remote_sha[:12]}`")
        else:
            lines.append(f"**First check**, recording upstream at `{remote_sha[:12]}`")
    _write_upstream_state({"sha": remote_sha, "checked_at": now.isoformat()})
    (REPO_ROOT / "UPSTREAM-DIFF.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    msg = f"upstream at {remote_sha[:12]}"
    if prev_sha and prev_sha != remote_sha:
        msg = f"upstream CHANGED: {prev_sha[:12]} -> {remote_sha[:12]}"
    return True, msg


def task_module_coverage() -> tuple[bool, str]:
    """Estimate KB module coverage against upstream source tree (without cloning)."""
    # Without a local clone, we use git ls-remote to get the tree, but
    # that only gives root commit. For a cheap estimate, count upstream Go
    # files via GitHub's API.
    api_url = f"https://api.github.com/repos/1Panel-dev/1Panel/git/trees/{UPSTREAM_BRANCH}?recursive=1"
    try:
        req = urllib.request.Request(api_url, headers={"User-Agent": "1panel-edu-research"})
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except Exception as e:
        return True, f"upstream tree API failed: {e}; skipping coverage"
    go_files = [t["path"] for t in data.get("tree", []) if t.get("path", "").endswith(".go")]
    # Categorize by top-level dir
    from collections import Counter
    cat = Counter()
    for p in go_files:
        top = p.split("/", 1)[0] if "/" in p else p
        cat[top] += 1
    total_go = len(go_files)
    # Map KB modules to upstream areas (heuristic from 00-landscape.md)
    coverage_map = {
        "01-app-store":    ["agent/app/service/app.go", "agent/app/service/app_install.go"],
        "02-container":     ["agent/app/service/container.go", "agent/app/service/image.go"],
        "03-website":       ["agent/app/service/website.go"],
        "04-database":      ["agent/app/service/database.go", "agent/app/service/database_*"],
        "05-backup-snapshot": ["agent/app/service/snapshot.go"],
        "06-cronjob":       ["agent/app/service/cronjob.go"],
        "07-alert":         ["agent/app/service/alert.go"],
        "08-file":          ["agent/app/service/file.go"],
        "09-ai-agent":      ["agent/app/service/ai.go"],
        "10-host-monitor":  ["agent/app/service/host.go"],
        "11-runtime-ai":    ["agent/app/service/ai_runtime.go"],
        "12-security":      ["agent/app/service/ssh.go", "agent/app/service/firewall.go"],
        "13-frontend":      ["frontend/src/"],
    }
    lines = [
        "# Module Coverage",
        "",
        f"Generated: {now_local().isoformat()}",
        f"Upstream: `{UPSTREAM_REPO}` @{UPSTREAM_BRANCH}",
        "",
        f"**Total Go files upstream**: {total_go}",
        "",
        "## By top-level dir",
        "",
        "| Top dir | .go files |",
        "|---|---|",
    ]
    for k, v in cat.most_common():
        lines.append(f"| {k} | {v} |")
    lines.extend([
        "",
        "## KB module coverage (heuristic)",
        "",
        "| KB module | Estimated upstream paths |",
        "|---|---|",
    ])
    for mod, paths in coverage_map.items():
        lines.append(f"| {mod} | {' / '.join(paths)} |")
    lines.append("")
    lines.append("> 注: 这是粗粒度映射, 真实覆盖率需要逐文件比对 HR.md 引用. 后续可升级.")
    (REPO_ROOT / "COVERAGE-REPORT.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    return True, f"{total_go} upstream .go files, 13 KB modules mapped"


def _read_kpi(path: Path, key: str) -> str | None:
    """Extract a key=value line from a simple markdown KPI file."""
    if not path.exists():
        return None
    try:
        for line in path.read_text(encoding="utf-8", errors="ignore").splitlines():
            line = line.strip()
            if line.lower().startswith(key.lower() + ":"):
                return line.split(":", 1)[1].strip()
    except OSError:
        return None
    return None


def task_health_score() -> tuple[bool, str]:
    """Compute 0-100 health score from quality / coverage / upstream / circuit breaker."""
    score = 100
    notes: list[str] = []

    # 1. Quality issues
    qr = REPO_ROOT / "QUALITY-REPORT.md"
    if qr.exists():
        text = qr.read_text(encoding="utf-8", errors="ignore")
        issue_count = sum(1 for line in text.splitlines() if line.startswith("- ") and "OK" not in line)
        if issue_count > 0:
            score -= min(issue_count * 3, 30)
            notes.append(f"Quality: {issue_count} issue(s) -> -{min(issue_count * 3, 30)}")

    # 2. Upstream stale
    ud = REPO_ROOT / "UPSTREAM-DIFF.md"
    if ud.exists():
        ut = ud.read_text(encoding="utf-8", errors="ignore")
        if "**Stale:" in ut:
            score -= 10
            notes.append("Upstream stale >= 7 days -> -10")

    # 3. Module coverage unknown -> -5
    cr = REPO_ROOT / "COVERAGE-REPORT.md"
    if not cr.exists():
        score -= 5
        notes.append("Coverage report missing -> -5")

    # 4. Circuit breaker exclusions
    state = load_state()
    cb = state.get("circuit_breaker", {})
    excluded = [tid for tid, e in cb.items() if e.get("excluded")]
    if excluded:
        score -= 20 * len(excluded)
        notes.append(f"CB excluded: {', '.join(excluded)} -> -{20 * len(excluded)}")

    # 5. Task failures this window
    cw = state.get("current_window", {})
    failed = sum(1 for t in cw.get("sub_tasks", []) if t.get("status") == "failed")
    if failed > 0:
        score -= 5 * failed
        notes.append(f"Window failures: {failed} -> -{5 * failed}")

    # Clamp
    score = max(0, min(100, score))
    if score >= 90:
        grade = "A"
    elif score >= 80:
        grade = "B"
    elif score >= 70:
        grade = "C"
    elif score >= 60:
        grade = "D"
    else:
        grade = "F"

    lines = [
        "# KB Health Score",
        "",
        f"Generated: {now_local().isoformat()}",
        "",
        f"## Score: **{score} / 100** (grade {grade})",
        "",
        "### Notes",
        "",
    ]
    if notes:
        lines.extend(f"- {n}" for n in notes)
    else:
        lines.append("- All checks passed, no deductions")
    (REPO_ROOT / "HEALTH-SCORE.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    return True, f"{score}/100 (grade {grade})"


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
    """git add + commit + push via pure Python subprocess. No .bat / .ps1 / cmd."""
    # 1. Check for changes
    try:
        r = subprocess.run(
            ["git", "-C", str(REPO_ROOT), "status", "--porcelain"],
            capture_output=True, text=True, timeout=10, check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as e:
        return False, f"git status failed: {e}"
    if not r.stdout.strip():
        return True, "no changes to sync"
    # 2. Add
    r = subprocess.run(
        ["git", "-C", str(REPO_ROOT), "add", "-A"],
        capture_output=True, text=True, timeout=30, check=False,
    )
    if r.returncode != 0:
        return False, f"git add failed: {r.stderr.strip()[:200]}"
    # 3. Commit
    msg = f"chore-daily-mgmt {now_local().strftime('%Y-%m-%d %H:%M')}"
    r = subprocess.run(
        ["git", "-C", str(REPO_ROOT), "commit", "-m", msg],
        capture_output=True, text=True, timeout=30, check=False,
    )
    if r.returncode != 0:
        return False, f"git commit failed: {r.stderr.strip()[:200]}"
    # 4. Push
    r = subprocess.run(
        ["git", "-C", str(REPO_ROOT), "push", "origin", "main"],
        capture_output=True, text=True, timeout=120, check=False,
    )
    if r.returncode != 0:
        return False, f"git push failed: {r.stderr.strip()[:200]}"
    # 5. Verify
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
        return True, "pushed but rev-parse failed"
    if local and remote and local == remote:
        return True, f"pushed {local[:7]}"
    return True, f"pushed; local={local[:7]} remote={remote[:7]} (verify later)"


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


def _is_sunday(now: datetime) -> bool:
    """ISO weekday: Mon=1..Sun=7. Sunday is 7."""
    return now.isoweekday() == 7


def task_daily_summary() -> tuple[bool, str]:
    """Summarize today's 5h windows. On Sunday, also write WEEKLY-REPORT.md."""
    state = load_state()
    history = state.get("history", [])
    today = now_local().date()
    today_iso = today.isoformat()
    todays = [h for h in history if h.get("completed_at", "").startswith(today_iso)]
    total_done = sum(h.get("tasks_done", 0) for h in todays)
    total_failed = sum(h.get("tasks_failed", 0) for h in todays)
    total_skipped = sum(h.get("tasks_skipped", 0) for h in todays)
    total_tasks = sum(h.get("total", 0) for h in todays)
    lines = [
        "# Daily Summary",
        "",
        f"Date: {today_iso}",
        f"Generated: {now_local().isoformat()}",
        "",
        f"## Today's totals",
        "",
        f"- Windows completed: {len(todays)}",
        f"- Tasks done: {total_done} / {total_tasks}",
        f"- Tasks failed: {total_failed}",
        f"- Tasks skipped (excluded by CB): {total_skipped}",
        "",
        "## Per-window breakdown",
        "",
        "| Window | done | failed | skipped |",
        "|---|---|---|---|",
    ]
    for h in todays:
        lines.append(f"| {h.get('window_id', '?')} | {h.get('tasks_done', 0)} | {h.get('tasks_failed', 0)} | {h.get('tasks_skipped', 0)} |")
    (REPO_ROOT / "DAILY-SUMMARY.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
    msg = f"{total_done}/{total_tasks} done, {total_failed} failed"
    # Sunday → weekly report
    if _is_sunday(now_local()):
        week_lines = [
            "# Weekly Report",
            "",
            f"Week ending: {today_iso}",
            f"Generated: {now_local().isoformat()}",
            "",
            f"## This week's totals",
            "",
            f"- Days processed: {len(set(h.get('completed_at', '')[:10] for h in history))}",
            f"- Total tasks done: {total_done}",
            f"- Total tasks failed: {total_failed}",
            "",
            "## All windows this week",
            "",
            "| Window | done | failed | skipped |",
            "|---|---|---|---|",
        ]
        for h in todays:
            week_lines.append(f"| {h.get('window_id', '?')} | {h.get('tasks_done', 0)} | {h.get('tasks_failed', 0)} | {h.get('tasks_skipped', 0)} |")
        (REPO_ROOT / "WEEKLY-REPORT.md").write_text("\n".join(week_lines) + "\n", encoding="utf-8")
        msg += " + weekly report"
    return True, msg


TASKS = {
    "index-rebuild":   task_index_rebuild,
    "quality-check":   task_quality_check,
    "upstream-poll":   task_upstream_poll,
    "module-coverage": task_module_coverage,
    "health-score":    task_health_score,
    "stats-report":    task_stats_report,
    "git-sync":        task_git_sync,
    "backup-snapshot": task_backup_snapshot,
    "daily-summary":   task_daily_summary,
}


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def run_task(state, task):
    task_id = task["id"]
    handler = TASKS.get(task_id)
    if handler is None:
        return False, f"no handler for {task_id}"
    try:
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

    # Anti-re-run: only pending/failed (and not excluded)
    next_task = find_next_task(state)
    if next_task is None:
        # All tasks done (no pending, no failed) → force regen to start next round.
        # This avoids 4.4h idle gap when 9 tasks complete in ~2.5h.
        all_done = all(t["status"] == "done" for t in state["current_window"]["sub_tasks"])
        if all_done:
            n = len(state["current_window"]["sub_tasks"])
            print(f"[{now_local().strftime('%H:%M')}] all {n} tasks done; force regen next round")
            subprocess.run(
                [sys.executable, str(SCHEDULER_DIR / "gen-plan.py")],
                check=True, timeout=30,
            )
            state = load_state()
            next_task = find_next_task(state)
    if next_task is None:
        total = len(state["current_window"]["sub_tasks"])
        print(f"<mavis-progress>silent: window {state['current_window']['window_id']} all done ({total}/{total})</mavis-progress>")
        return 0

    task_id = next_task["id"]

    # Acquire lock
    if not acquire_lock(task_id):
        return 0  # skip silently, lock held

    try:
        mark_running(state, task_id)
        save_state(state)

        start_time = time.time()
        ts = now_local().strftime("%H:%M:%S")
        print(f"[{ts}] running: {task_id} ({next_task['name']})")
        ok, msg = run_task(state, next_task)
        duration = int(time.time() - start_time)

        # Update CB and state
        on_task_complete(state, task_id, ok, msg)
        finalize(state, task_id, ok, msg, duration)
        save_state(state)

        # Alert on failure
        if not ok:
            append_alert(task_id, msg)

        ts = now_local().strftime("%H:%M:%S")
        status = "done" if ok else "failed"
        print(f"[{ts}] {status}: {task_id} ({msg}) [{duration}s]")
        if ok:
            print(f"<mavis-progress>tick: {task_id} {msg} [{duration}s]</mavis-progress>")
            return 0
        print(f"<mavis-progress>tick: {task_id} FAILED: {msg}</mavis-progress>")
        return 1
    finally:
        release_lock()


if __name__ == "__main__":
    sys.exit(main())
