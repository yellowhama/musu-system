# Sample Wiki Fixture

This fixture gives `musu-marketer` a tiny grounded wiki for smoke checks.

Suggested usage:

```powershell
./musu-marketer.exe doctor --project demo --wiki .\examples\sample-wiki --topic scheduler --json
./musu-marketer.exe draft "scheduler" --project demo --wiki .\examples\sample-wiki
```

Files:
- `index.json`: minimal page metadata
- `topics/scheduler-overview.md`: grounded scheduler topic content
