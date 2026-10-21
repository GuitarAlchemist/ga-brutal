# Extraction validation

Validated on Windows with .NET SDK 10.0.113. Source: GA #839 at `1c5781daed923ffc3104296554bbe956a16d3758`.
Before validation: 28,467 MB available RAM, 93.98 GiB committed, 33.43 GiB commit headroom.

| Project | Locked restore | Serial build | Warnings | Errors |
|---|---|---|---:|---:|
| HelloBrutal | PASS | PASS | 0 | 0 |
| HallSlice | PASS | PASS | 0 | 0 |
| Colossus | PASS | PASS | 0 | 0 |
| Tour | PASS | PASS | 2 | 0 |

Commands, one project at a time:

```powershell
dotnet restore <project.csproj> --locked-mode --disable-parallel -v:minimal
dotnet build <project.csproj> --no-restore -m:1 -p:BuildInParallel=false -p:UseSharedCompilation=false -v:minimal
python scripts/verify_snapshot.py
```

Tour retains two source warnings: CS8604 at Program.cs:149 and BRUTALCORE0011 at Program.cs:457.
GA's warning/advisory suppression files were not inherited. Original locks remain byte-identical.
The integrity checker verifies included files against their recorded hashes, excluded assets stay absent,
all four projects have no GA project references/imports, eight textures are present, Python source parses,
and known credential/private-key patterns are absent. This pattern scan is not a universal secret detector.

No GPU/window/audio/runtime/streaming test was run. No new renderer or stream process was started.
Existing screenshots/reports are historical upstream evidence, not new validation results.
Local build/restore logs and generated bin/obj are ignored; VALIDATION.json retains each log's SHA-256.
No binary release, official logo startup or complete native-distribution compliance is claimed.
