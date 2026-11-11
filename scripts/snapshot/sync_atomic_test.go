package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncLateExtractionFailureLeavesSnapshotUnchanged(t *testing.T) {
	root := copySnapshot(t)
	ga := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", ga}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %s: %v", out, err)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.invalid")
	git("config", "user.name", "Snapshot test")
	// This sorts before Program.cs. The old implementation overwrites it before
	// the missing extraction pattern in the later file is discovered.
	first := "Experiments/Brutal/Colossus/City.cs"
	late := "Experiments/Brutal/Colossus/Program.cs"
	for path, text := range map[string]string{first: "changed upstream\n", late: "extraction patterns changed\n"} {
		target := filepath.Join(ga, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-qm", "fixture")
	before, err := os.ReadFile(filepath.Join(root, first))
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := os.ReadFile(filepath.Join(root, "PROVENANCE.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = syncSnapshot(ga, "HEAD", root)
	if err == nil || !strings.Contains(err.Error(), "expected one match") {
		t.Fatalf("want extraction failure, got %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(root, first))
	if !bytes.Equal(before, after) {
		t.Fatal("late validation failure modified an earlier source")
	}
	after, _ = os.ReadFile(filepath.Join(root, "PROVENANCE.json"))
	if !bytes.Equal(provenance, after) {
		t.Fatal("failed sync modified provenance")
	}
	if _, err := verify(root); err != nil {
		t.Fatalf("failed sync invalidated snapshot: %v", err)
	}
}

func TestPublishRollsBackEarlierWritesAndRemovals(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	removed := filepath.Join(root, "removed")
	blocker := filepath.Join(root, "parent-file")
	for _, path := range []string{first, removed} {
		if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// The impossible parent is encountered after an overwrite and a deletion.
	err := publishSnapshot(root, []snapshotOutput{
		{path: first, data: []byte("new")}, {path: removed, remove: true},
		{path: blocker, data: []byte("new parent file")},
		{path: filepath.Join(blocker, "child"), data: []byte("late")},
	})
	if err == nil {
		t.Fatal("want publication failure")
	}
	for _, path := range []string{first, removed} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "original" {
			t.Fatalf("rollback %s: %q, %v", path, data, err)
		}
	}
	if _, err := os.Stat(blocker); !os.IsNotExist(err) {
		t.Fatalf("rollback left newly created parent: %v", err)
	}
}

func TestPublishWritesAndRemoves(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old")
	if err := os.WriteFile(old, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(root, "nested", "new")
	if err := publishSnapshot(root, []snapshotOutput{{path: old, remove: true}, {path: fresh, data: []byte("new")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old survives: %v", err)
	}
	data, err := os.ReadFile(fresh)
	if err != nil || string(data) != "new" {
		t.Fatalf("new: %q %v", data, err)
	}
}

func TestTrafficExtractionEditIsExact(t *testing.T) {
	const path = "Experiments/Brutal/Colossus/Program.cs"
	e := edits[path][0]
	corrected, err := e.apply(e.old, path)
	if err != nil || corrected != e.new {
		t.Fatalf("traffic extraction: %q %v", corrected, err)
	}
	if _, err := e.apply(e.old+e.old, path); err == nil {
		t.Fatal("duplicated upstream budget must fail closed")
	}
	if _, err := e.apply("changed upstream budget", path); err == nil {
		t.Fatal("changed upstream budget must fail closed")
	}
	data, err := os.ReadFile(filepath.Join(repo, path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), e.new) != 1 {
		t.Fatal("snapshot does not contain exactly the recorded traffic edit")
	}
}

func TestSyncRefusesConcurrentWriter(t *testing.T) {
	root := t.TempDir()
	lock := filepath.Join(root, ".snapshot-sync.lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := syncSnapshot("unused", "HEAD", root); err == nil || !strings.Contains(err.Error(), "cannot acquire snapshot sync lock") {
		t.Fatalf("concurrent writer: %v", err)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("other writer's lock changed: %v", err)
	}
}
