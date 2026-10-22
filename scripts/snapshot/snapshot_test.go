package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests run in this folder; the snapshot is two levels up.
const repo = "../.."

func TestVerifyPassesOnTheSnapshot(t *testing.T) {
	if _, err := verify(repo); err != nil {
		t.Fatal(err)
	}
}

// A sync rewrites PROVENANCE.json: reading and writing it unchanged must give back the same bytes.
func TestProvenanceRoundTrips(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repo, "PROVENANCE.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := loadProvenance(repo)
	if err != nil {
		t.Fatal(err)
	}
	out, err := p.encode()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Fatal("encoding PROVENANCE.json changes its bytes")
	}
}

// A checker that never failed proves nothing: each case breaks a copy of the snapshot in the way a check exists for.
func TestVerifyCatches(t *testing.T) {
	p, err := loadProvenance(repo)
	if err != nil {
		t.Fatal(err)
	}
	var source, left string
	for _, row := range p.Files {
		if row.Included && source == "" && strings.HasSuffix(row.Path, ".fs") {
			source = row.Path
		}
		if !row.Included && left == "" {
			left = row.Path
		}
	}
	write := func(path, text string) func(t *testing.T, root string) {
		return func(t *testing.T, root string) {
			target := filepath.Join(root, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	cases := []struct {
		name   string
		change func(t *testing.T, root string)
		want   string
	}{
		{"an edited source file", write(source, "// edited\n"), source + ": SHA-256"},
		{"an excluded asset back in the tree", write(left, "baked"), "excluded upstream data appeared"},
		{"a token in a new file", write("notes.md", "ghp_"+strings.Repeat("a", 36)), "secret pattern match in notes.md"},
		{"an .env file", write("Experiments/.env.local", "KEY=1"), ".env file"},
		{"a blocked binary", write("tools/helper.dll", "MZ"), "blocked file type"},
		// Recorded in PROVENANCE.json as a sync would record it, so that only the boundary check stands in the way.
		{"a GA project reference", func(t *testing.T, root string) {
			const tour = "Experiments/Brutal/Tour/Tour.csproj"
			target := filepath.Join(root, filepath.FromSlash(tour))
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(data), "</Project>",
				`<ItemGroup><ProjectReference Include="..\..\..\Common\GA.Core\GA.Core.csproj" /></ItemGroup></Project>`, 1)
			if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			p, err := loadProvenance(root)
			if err != nil {
				t.Fatal(err)
			}
			for i := range p.Files {
				if p.Files[i].Path == tour {
					p.Files[i].ExtractedSHA256, _ = sha256File(target)
				}
			}
			out, _ := p.encode()
			if err := os.WriteFile(filepath.Join(root, "PROVENANCE.json"), out, 0o644); err != nil {
				t.Fatal(err)
			}
		}, "references ../../../Common/GA.Core/GA.Core.csproj"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := copySnapshot(t)
			c.change(t, root)
			_, err := verify(root)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want an error containing %q", err, c.want)
			}
		})
	}
}

func copySnapshot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(repo, path)
		if d.IsDir() {
			if name := d.Name(); path != repo && (name == ".git" || name == "bin" || name == "obj") {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}
