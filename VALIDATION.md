# Extraction validation

Validated on Windows with .NET SDK 10.0.113. Source: GA #840 at `644e25db913a53f43b1464677fd130fe79c75090`, which builds on GA #839.
`VALIDATION.json` keeps the log hashes of the run at `76efbffd` (10,169 MB available RAM before it); the runs since
kept no logs, and CI repeats them on each pull request.

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
dotnet run -c Release --project Experiments/Brutal/Lego.Cli -p:RestoreLockedMode=true -- test
go run ./scripts/snapshot verify
go -C Experiments/Brutal/Colossus/comfyui test .
```

Tour retains two source warnings: CS8604 at Program.cs:149 and BRUTALCORE0011 at Program.cs:457.
GA's warning/advisory suppression files were not inherited. Original locks remain byte-identical.

The Lego tests run on the CPU and draw nothing. In Release, with the locked restore, all 21 pass and the locks stay
unchanged. In Debug, 20 of 21 pass: "describing again and again keeps nothing" kept 6,362,560 bytes, because unoptimised
code keeps locals alive. Run from outside the repository, `dotnet` ignored `global.json` and chose the host's newer SDK,
11.0.100-preview.3; its restore rewrote the two Lego locks with FSharp.Core 11.0.100-preview3.26207.106. They were
restored from the source blobs, and the checker passed again. CI runs every command above on `windows-latest` with
exactly SDK 10.0.113.

The integrity checker, `scripts/snapshot` in Go with its standard library only, verifies included files against their
recorded hashes, excluded assets stay absent, all six projects have no GA project references/imports, twelve textures
are present, every text file is UTF-8, and known credential/private-key patterns are absent. This pattern scan is not a
universal secret detector. Its tests break a copy of the snapshot in six of those ways, and each must fail.
`go run ./scripts/snapshot sync <GA checkout> <commit>` brings the snapshot up to a newer GA commit by the same rules:
it reads git blobs, never a working tree, and each extraction edit must match exactly once.

No GPU/window/audio/streaming test was run. No new renderer or stream process was started.
Existing screenshots/reports are historical upstream evidence, not new validation results.
Local build/restore logs and generated bin/obj are ignored; VALIDATION.json retains each log's SHA-256.
No binary release, official logo startup or complete native-distribution compliance is claimed.