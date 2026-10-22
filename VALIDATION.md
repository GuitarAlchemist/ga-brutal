# Extraction validation

Validated on Windows with .NET SDK 10.0.113. Source: GA #840 at `76efbffd5af82b46c3146957a6bd4504008eeb89`, which builds on GA #839.
Before validation: 10,169 MB available RAM.

| Project | Language | Locked restore | Serial build | Warnings | Errors |
|---|---|---|---|---:|---:|
| HelloBrutal | C# | PASS | PASS | 0 | 0 |
| HallSlice | C# | PASS | PASS | 0 | 0 |
| Colossus | C# | PASS | PASS | 0 | 0 |
| Tour | C# | PASS | PASS | 2 | 0 |
| Lego | F# | PASS | PASS | 0 | 0 |
| Lego.Cli | F# | PASS | PASS | 0 | 0 |

Commands, one project at a time:

```powershell
dotnet restore <project> --locked-mode --disable-parallel -v:minimal
dotnet build <project> --no-restore -m:1 -p:BuildInParallel=false -p:UseSharedCompilation=false -v:minimal
dotnet run --no-restore --project Experiments/Brutal/Lego.Cli -- test
python scripts/verify_snapshot.py
```

Tour retains two source warnings: CS8604 at Program.cs:149 and BRUTALCORE0011 at Program.cs:457.
GA's warning/advisory suppression files were not inherited. Original locks remain byte-identical.

The Lego tests run on the CPU and draw nothing. In Debug, 19 of 20 pass: "describing again and again keeps nothing"
kept 6,362,584 bytes, because unoptimised code keeps locals alive. In Release all 20 pass, but a Release restore
resolves FSharp.Core 11.0.100-preview3.26207.106 and rewrites the two Lego locks; they were restored from the source
blobs, and the checker passed again afterwards.

The integrity checker verifies included files against their recorded hashes, excluded assets stay absent,
all six projects have no GA project references/imports, twelve textures are present, Python source parses,
and known credential/private-key patterns are absent. This pattern scan is not a universal secret detector.

No GPU/window/audio/streaming test was run. No new renderer or stream process was started.
Existing screenshots/reports are historical upstream evidence, not new validation results.
Local build/restore logs and generated bin/obj are ignored; VALIDATION.json retains each log's SHA-256.
No binary release, official logo startup or complete native-distribution compliance is claimed.