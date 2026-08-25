"""Reset state for round 2: fix stale index-rebuild 'running' to 'pending', release stale lock."""
import json
import os
from pathlib import Path

sf = Path(r"D:\MiniMax Code\1Panel-edu-research\.scheduler\state.json")
lf = Path(r"D:\MiniMax Code\1Panel-edu-research\.scheduler\lock")

s = json.loads(sf.read_text(encoding="utf-8-sig"))
# Fix stale running
fixed = 0
for t in s["current_window"]["sub_tasks"]:
    if t["status"] == "running":
        t["status"] = "pending"
        t["started_at"] = None
        fixed += 1
sf.write_text(json.dumps(s, ensure_ascii=False, indent=2), encoding="utf-8")
print(f"fixed {fixed} stale 'running' -> 'pending'")

# Release stale lock
if lf.exists():
    lf.unlink()
    print(f"removed stale lock: {lf}")
else:
    print("no lock file")
