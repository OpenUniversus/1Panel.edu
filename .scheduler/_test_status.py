"""Test helper: print state.json summary."""
import json
from pathlib import Path

sf = Path(r"D:\MiniMax Code\1Panel-edu-research\.scheduler\state.json")
s = json.loads(sf.read_text(encoding="utf-8-sig"))
print(f"Window: {s['current_window']['window_id']}")
for t in s['current_window']['sub_tasks']:
    dur = t.get('duration_sec')
    dur_s = f"{dur}s" if dur is not None else "-"
    print(f"  [{t['status']:>7}] {t['id']:<18} ({dur_s:>4}) {t.get('result') or ''}")
print(f"\nHistory entries: {len(s.get('history', []))}")
print(f"Circuit breaker: {s.get('circuit_breaker', {})}")
print(f"Schema: {s.get('schema_version')}")
print(f"Lock file: {Path(r'D:\MiniMax Code\1Panel-edu-research\.scheduler\lock').exists()}")
