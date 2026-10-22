package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// The extraction's rules, shared by sync and verify.
const subtree = "Experiments/Brutal/"

var blocked = map[string]bool{
	".exe": true, ".dll": true, ".pdb": true, ".wav": true, ".onnx": true, ".safetensors": true, ".gguf": true,
	".pt": true, ".pth": true, ".bin": true, ".glb": true, ".mp4": true,
}

const excludedReason = "Optional baked data, recordings or screenshots excluded from this source-only snapshot; " +
	"retain upstream provenance."

// excluded tells the files the snapshot leaves out: baked data, recordings, and the screenshots kept as evidence.
func excluded(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	screenshot := strings.HasPrefix(path, subtree+"evidence/") && (ext == ".png" || ext == ".jpg" || ext == ".jpeg")
	return blocked[ext] || screenshot
}

// edit is one change of the extraction. It must match exactly once in the source blob. A pattern stands in for the old
// text where that text names a path on the source machine, so that no such path is written here either.
type edit struct {
	old     string
	pattern *regexp.Regexp
	new     string
}

var edits = map[string][]edit{
	"Experiments/Brutal/Colossus/Colossus.csproj": {
		{old: `<None Include="head\head_points.bin" CopyToOutputDirectory="PreserveNewest" />`,
			new: `<None Include="head\head_points.bin" Condition="Exists('head\head_points.bin')" CopyToOutputDirectory="PreserveNewest" />`},
		{old: `<None Include="city\towers.glb" CopyToOutputDirectory="PreserveNewest" />`,
			new: `<None Include="city\towers.glb" Condition="Exists('city\towers.glb')" CopyToOutputDirectory="PreserveNewest" />`},
	},
	"Experiments/Brutal/Colossus/Program.cs": {
		{old: `// --no-voice (don't mirror the voice plugins), --face-file PATH`,
			new: `// --voice-mirror (opt in to machine-local voice plugins), --no-voice (disable mirroring), --face-file PATH`},
		{old: `bool mirrorOn = !args.Contains("--no-voice") && !recording,`,
			new: `bool mirrorOn = args.Contains("--voice-mirror") && !args.Contains("--no-voice") && !recording,`},
		{old: `            dlss = Dlss.Load(ArgText("--sl-path") ?? Dlss.DefaultFolder, args.Contains("--sl-verbose"), rayAsked, fgAsked, out var why);
            dlssNote = why;`,
			new: `            // Extraction boundary: Streamline's folder must be given; no source-machine path is assumed.
            var slPath = ArgText("--sl-path");
            var why = "off (--sl-path DIR required: Streamline 2.14.1's bin, not in this repository)";
            dlss = slPath is null ? null : Dlss.Load(slPath, args.Contains("--sl-verbose"), rayAsked, fgAsked, out why);
            dlssNote = why;`},
	},
	"Experiments/Brutal/Colossus/Streamline.cs": {
		{pattern: regexp.MustCompile(`(?m)^    public const string DefaultFolder = @"[^"\n]*";\n`)},
		{old: "(--sl-path; by default the one the Observatory's\n// RTX build fetches from NVIDIA's own release)",
			new: "(--sl-path, which this extraction\n// requires; Streamline 2.14.1's bin from NVIDIA's own release)"},
	},
	"Experiments/Brutal/README.md": {
		{pattern: regexp.MustCompile("`--sl-path`, by default\n  `[^`\n]*`, Streamline 2\\.14\\.1's `bin` that the Observatory's RTX build fetched\\."),
			new: "`--sl-path DIR`, which\n  this extraction requires, Streamline 2.14.1's `bin` from NVIDIA's own release."},
	},
}

// kept files are unchanged upstream since the first extraction: their edited copy stays as it is.
var kept = map[string]bool{"Experiments/Brutal/Colossus/Speech.cs": true}

// editedPaths lists every file the extraction changes, sorted.
func editedPaths() []string {
	var paths []string
	for path := range edits {
		paths = append(paths, path)
	}
	for path := range kept {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}

func (e edit) apply(text, path string) (string, error) {
	if e.pattern != nil {
		if n := len(e.pattern.FindAllStringIndex(text, -1)); n != 1 {
			return "", fmt.Errorf("%s: expected one match for %q, found %d", path, e.pattern, n)
		}
		return e.pattern.ReplaceAllLiteralString(text, e.new), nil
	}
	if n := strings.Count(text, e.old); n != 1 {
		return "", fmt.Errorf("%s: expected one match for %.60q, found %d", path, e.old, n)
	}
	return strings.Replace(text, e.old, e.new, 1), nil
}

// syncSnapshot brings the snapshot in root up to a newer GA commit by the first extraction's rules: it reads immutable
// git blobs, never a working tree; keeps source, licences and texture PNGs; leaves out what excluded names; and applies
// the extraction's edits. It writes the files and PROVENANCE.json.
func syncSnapshot(ga, commit, root string) (string, error) {
	git := func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", append([]string{"-C", ga}, args...)...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(stderr.Bytes()))
		}
		return out, nil
	}
	old, err := loadProvenance(root)
	if err != nil {
		return "", err
	}
	oldRows := map[string]fileRow{}
	for _, row := range old.Files {
		oldRows[row.Path] = row
	}
	listing, err := git("ls-tree", "-r", "-l", "-z", commit, "--", subtree)
	if err != nil {
		return "", err
	}
	var rows []fileRow
	seen := map[string]bool{}
	edited := 0
	for _, entry := range strings.Split(strings.TrimSuffix(string(listing), "\x00"), "\x00") {
		meta, path, _ := strings.Cut(entry, "\t")
		fields := strings.Fields(meta) // mode, type, blob, size
		if len(fields) != 4 || fields[1] != "blob" || fields[0] == "120000" {
			return "", fmt.Errorf("%s: not a plain file (%s)", path, meta)
		}
		blob := fields[2]
		data, err := git("cat-file", "blob", blob)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		row := fileRow{Path: path, GitBlob: blob, SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(data))}
		seen[path] = true
		target := filepath.Join(root, filepath.FromSlash(path))
		if excluded(path) {
			if _, known := oldRows[path]; !known && exists(target) {
				return "", fmt.Errorf("%s is in the tree but was never part of the snapshot", path)
			}
			if err := removeIfPresent(target); err != nil {
				return "", err
			}
			row.Reason = excludedReason
			rows = append(rows, row)
			continue
		}
		out, modified := data, false
		switch {
		case kept[path]:
			if oldRows[path].GitBlob != blob {
				return "", fmt.Errorf("%s changed upstream since the first extraction: redo its edit", path)
			}
			if out, err = os.ReadFile(target); err != nil {
				return "", err
			}
			modified = true
		case edits[path] != nil:
			text := string(data)
			for _, e := range edits[path] {
				if text, err = e.apply(text, path); err != nil {
					return "", err
				}
			}
			out, modified = []byte(text), true
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, out, 0o644); err != nil {
			return "", err
		}
		if modified {
			edited++
		}
		sum = sha256.Sum256(out)
		row.Included, row.ExtractedSHA256, row.Modified = true, hex.EncodeToString(sum[:]), &modified
		rows = append(rows, row)
	}

	// Files of the old snapshot that the new commit no longer has.
	var gone []string
	for _, row := range old.Files {
		if !seen[row.Path] {
			gone = append(gone, row.Path)
			if err := removeIfPresent(filepath.Join(root, filepath.FromSlash(row.Path))); err != nil {
				return "", err
			}
		}
	}

	head, err := git("rev-parse", commit)
	if err != nil {
		return "", err
	}
	tree, err := git("rev-parse", commit+":"+strings.TrimSuffix(subtree, "/"))
	if err != nil {
		return "", err
	}
	next := *old
	next.SourceCommit = strings.TrimSpace(string(head))
	next.SourceTree = strings.TrimSpace(string(tree))
	if next.SourceCommit != old.SourceCommit {
		next.PreviousSourceCommit = old.SourceCommit
	}
	next.OwnerAcknowledgement = "Synced by the owner session at the author's request on " +
		time.Now().Format("2006-01-02") + ". No source checkout files were changed."
	next.Files = rows
	data, err := next.encode()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(root, "PROVENANCE.json"), data, 0o644); err != nil {
		return "", err
	}
	included := 0
	for _, row := range rows {
		if row.Included {
			included++
		}
	}
	return fmt.Sprintf("SYNC %s: %d files, %d included, %d excluded, %d edited, %d gone: %v",
		next.SourceCommit[:8], len(rows), included, len(rows)-included, edited, len(gone), gone), nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
