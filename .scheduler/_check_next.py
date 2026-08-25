"""One-shot: print next 5 sub-tick wakeups relative to now."""
from datetime import datetime, timedelta, timezone

TZ = timezone(timedelta(hours=8))
now = datetime.now(TZ)
print(f"now: {now.strftime('%H:%M:%S')}")

# Build candidates: next :15, :30, :45, next-hour-:00 (or master)
candidates = []
for m in (15, 30, 45):
    if m > now.minute:
        nt = now.replace(minute=m, second=0, microsecond=0)
        candidates.append((nt, "sub"))
        break
else:
    nt = (now + timedelta(hours=1)).replace(minute=0, second=0, microsecond=0)
    candidates.append((nt, "sub/master"))

# next master
bucket = (now.hour // 5) * 5
next_master = bucket + 5 if (now.hour // 5) * 5 + 5 <= 20 else 0
m_at = now.replace(hour=next_master, minute=0, second=0, microsecond=0)
if m_at <= now:
    m_at = (now + timedelta(days=1)).replace(hour=0, minute=0, second=0, microsecond=0)
candidates.append((m_at, "master"))

# Show next 5
candidates.sort()
for ts, kind in candidates[:5]:
    delta = (ts - now).total_seconds()
    print(f"  {ts.strftime('%H:%M')} [{kind}] in {delta:.0f}s")
