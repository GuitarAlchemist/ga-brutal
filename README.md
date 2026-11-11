# GA BRUTAL

Private, standalone **source evaluation** of BRUTAL and the Colossus scene. Coordination remains in
[Gaia #257](https://github.com/GuitarAlchemist/gaia/issues/257).

This is a verified snapshot of [GA #840](https://github.com/GuitarAlchemist/ga/pull/840), which builds on
[GA #839](https://github.com/GuitarAlchemist/ga/pull/839), at
[`76efbffd5af82b46c3146957a6bd4504008eeb89`](https://github.com/GuitarAlchemist/ga/commit/76efbffd5af82b46c3146957a6bd4504008eeb89).
It was first extracted from #839 at `1c5781daed923ffc3104296554bbe956a16d3758`. GA and its PRs/history are preserved.
No merge, deployment, stream restart or owner acknowledgement is claimed. Observatory's checkout is untouched.

## Contents

Six .NET 10 projects are under `Experiments/Brutal/`: `HelloBrutal`, `HallSlice`, `Colossus` and `Tour` in C#, and
the F# `Lego` library with its `Lego.Cli`. `HallSlice` and `Colossus` reference `Lego`; no project references GA.
They keep their original lock files, C#/F#/Python/PowerShell sources and historical reports.
The twelve documented Colossus texture PNGs are included unchanged. `PROVENANCE.json` records every source
blob and SHA-256, including excluded assets; `SOURCE_HISTORY.md` links the source commits.

Recorded WAVs, baked `head_points.bin`, `towers.glb`, screenshots (the evidence PNGs and JPGs), engines, native DLLs, bin/obj,
models and caches are excluded. Colossus has procedural head/city fallbacks. The ten recorded WAVs have
documented Chatterbox/Kyutai sources, but their generation-to-byte provenance was not independently verified
for this extraction; their source hashes are retained rather than redistributing them.

## Build and use

SDK: `global.json` pins 10.0.104 with `latestFeature` roll-forward (validation host has 10.0.113).
Each project has exact direct package pins and a NuGet lock. No project references GA. The new root build
files intentionally exclude GA's warning/advisory suppressions, version overrides and blue/green slots.
The locks retain `Microsoft.Extensions.ObjectPool` 11.0.0-preview.5.26302.115 where selected upstream;
this is not represented as a stable .NET 10-only dependency graph.
The F# locks pin the FSharp.Core that the SDK brings: 10.0.113, with SDK 10.0.113. Run `dotnet` from the repository
root, so that `global.json` chooses the SDK. Run from elsewhere, a newer SDK brings its own FSharp.Core, and restore rewrites
the Lego locks. CI (`.github/workflows/ci.yml`) installs exactly 10.0.113 and runs the checks below on each pull request.

```powershell
dotnet restore Experiments/Brutal/Colossus/Colossus.csproj --locked-mode --disable-parallel
dotnet build Experiments/Brutal/Colossus/Colossus.csproj --no-restore -m:1
dotnet run -c Release --project Experiments/Brutal/Lego.Cli -p:RestoreLockedMode=true -- test   # the Lego logic, no engine
# Optional Windows runtime, only when GPU/memory/streaming capacity permits:
dotnet run --no-build --project Experiments/Brutal/Colossus -- --no-tts --no-voice --no-sound --no-rtx --swarm 1000
```

No graphical, audio, GPU or live streaming run was performed for extraction validation.
Historical results in `Experiments/Brutal/README.md` describe the original author's machine and original
asset set, not fresh results for this repository. Consult `VALIDATION.md` for this extraction's checks.

## Optional integrations and asset provenance

- NVIDIA DLSS, Ray Reconstruction and frame generation (`--dlss`, `--rr`, `--fg`) need Streamline 2.14.1's
  NVIDIA-signed DLLs from an explicit `--sl-path DIR`. None is included, and no machine path is assumed.
  At run time, Streamline may load newer NVIDIA-signed plugins from the driver's own NGX store.
- Machine-local voice mirroring now requires `--voice-mirror`. No voice plugins are bundled or started.
- Piper TTS requires **both** explicit `--piper` and `--voice` paths with independently suitable licences.
  The source machine's SIWIS/lessac model is never auto-selected. No model or old recordings are included.
- FFmpeg, Piper, Blender, ComfyUI and their models/workflows remain external, optional operator tools.
  No download or licence-acceptance step is automated. Segoe UI is read from the installed Windows fonts,
  if present; no font is bundled.
- `Colossus/comfyui` (Go) makes the textures from an external workflow, learn-visuals' `09-seamless.api.json`, given
  with `-workflow`; no machine path is assumed.
  That workflow was not audited/copied; generation is **not self-contained**. Existing twelve texture bytes
  and their documented seeds/model provenance are preserved; no generator was executed.
- The optional head-generation source documents TRELLIS.2 and DINOv3: **Built with DINOv3**.
  The pipeline is not labelled entirely MIT. Its baked output and all model weights are excluded.

## Licences and release boundary

GA-origin source retains the original [MIT licence, copyright 2021 GuitarAlchemist](LICENSE).
BRUTAL is **not relabelled MIT**: its separate [BRUTAL License v1.0](LICENSES/BRUTAL/LICENSE.txt)
and [NOTICE](NOTICE.txt) are retained at upstream revision `a25cdf80b2a15530e7da6cff744def9d841ad3c3`.
OpenAL Soft is LGPL-2.0-or-later; FreeType carries FTL terms. Package-specific licence texts are under
`LICENSES/`; see `THIRD_PARTY.md`. FMOD is excluded.

**This repository is not a distribution-ready binary product.** A distributed BRUTAL product requires
the full copyright/credits, the official unaltered logo before the primary function, and its NOTICE,
plus all relevant transitive/native notices and obligations. The logo/startup sequence and complete
binary release compliance are not validated here. No engine/native binaries or release archives are published.
No new explicit third-party agreement has been accepted.
