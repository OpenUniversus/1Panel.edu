"""Test helper: reset state.json in-place (no file deletion needed)."""
import json
import sys
from pathlib import Path

sf = Path(r"D:\MiniMax Code\1Panel-edu-research\.scheduler\state.json")
s = json.loads(sf.read_text(encoding="utf-8-sig"))
s["current_window"] = None
s["history"] = []
s["circuit_breaker"] = {}
sf.write_text(json.dumps(s, ensure_ascii=False, indent=2), encoding="utf-8")
print("state.json reset: current_window=None")
