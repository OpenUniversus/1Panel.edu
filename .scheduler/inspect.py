#!/usr/bin/env python3
"""Quick state inspector for the 1Panel.edu scheduler."""
import json
import sys
from pathlib import Path

state_file = Path(__file__).parent / "state.json"
state = json.loads(state_file.read_text(encoding="utf-8-sig"))
w = state["current_window"]
print(f"window: {w['window_id']}")
print(f"start:  {w['window_start']}")
print(f"end:    {w['window_end']}")
print()
for t in w["sub_tasks"]:
    dur = t.get("duration_sec")
    dur_s = f" ({dur}s)" if isinstance(dur, int) else ""
    print(f"  [{t['id']:>16}] {t['status']:<10}{dur_s}  {t['name']}")
if state.get("history"):
    print()
    print("history:")
    for h in state["history"][-5:]:
        print(f"  {h['window_id']}: {h['tasks_done']} done / {h['tasks_failed']} failed")
