# Third-party and asset boundaries

## BRUTAL

Source/licence audit reference: https://github.com/ahwootech/BRUTAL/tree/a25cdf80b2a15530e7da6cff744def9d841ad3c3
The source snapshot depends on Ahwoo NuGet packages; no engine or native package is vendored.
`LICENSES/BRUTAL/LICENSE.txt` and `NOTICE.txt` retain BRUTAL's terms and copyright, distinct from GA's MIT source licence.
The distributed-product logo/credits/NOTICE requirements remain an open release gate, not an implemented feature.

## Native dependencies

- Ahwoo.Brutal.OpenAl 2026.10.2: OpenAL Soft 1.25.2, LGPL-2.0-or-later; its package-specific COPYING and
  third-party notices are copied as text under `LICENSES/ahwoo.brutal.openal/`. A binary release needs
  separate validation of source/relinking and applicable notices; no such release is made here.
- Ahwoo.Brutal.FreeType 2026.10.1: the FTL and package notice are under `LICENSES/ahwoo.brutal.freetype/`.
- Other BRUTAL native/transitive components retain their own per-package licences. Lock files record the
  full dependency identities. The repository does not claim that this partial text inventory is a complete
  binary-distribution notice bundle. FMOD is not referenced.

## Assets and optional software

Eight texture PNGs retain the source README's documented local Z-Image-Turbo/Qwen/Blender provenance and hashes.
No model, workflow or font is shipped. Baked head/city data and all WAV/screenshot bytes are excluded;
their original Git blob IDs and hashes remain in `PROVENANCE.json`.
The voice README is retained as historical provenance, not a representation that its recordings are included.
Chatterbox MIT and Kyutai voice-donation CC0 declarations do not establish the omitted WAVs' generation-to-byte chain.
Piper, SIWIS/lessac voices, FFmpeg, Windows fonts, ComfyUI and Blender remain external optional tools, with
their own licence review before use. Nothing here accepts an agreement or downloads them.

Generated assets and tools are not all MIT simply because the surrounding source is MIT. In particular,
the source's optional TRELLIS.2/DINOv3 workflow has DINOv3-specific terms and attribution.
