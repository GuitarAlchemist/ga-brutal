package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type snapshotOutput struct {
	path   string
	data   []byte
	remove bool
}

// All Git reads, extraction edits and provenance encoding finish before this
// function is called. Stage every byte before publishing. On a reported I/O
// failure restore already replaced files, including provenance. This is not a
// crash-safe transaction or a lock against concurrent writers/readers.
func publishSnapshot(root string, output []snapshotOutput) (err error) {
	stage, err := os.MkdirTemp(filepath.Dir(root), ".snapshot-stage-")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.RemoveAll(stage)
		}
	}()
	for i, item := range output {
		if info, e := os.Lstat(item.path); e == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("refuse non-regular snapshot target: %s", item.path)
		} else if e != nil && !errors.Is(e, fs.ErrNotExist) {
			return e
		}
		if !item.remove {
			if err := os.WriteFile(filepath.Join(stage, fmt.Sprintf("new-%d", i)), item.data, 0644); err != nil {
				return err
			}
		}
	}
	type applied struct {
		target, backup string
		installed      bool
	}
	var done []applied
	defer func() {
		if err == nil {
			return
		}
		for i := len(done) - 1; i >= 0; i-- {
			item := done[i]
			if item.installed {
				if e := os.Remove(item.target); e != nil {
					err = errors.Join(err, e)
					keepBackup = true
				}
			}
			if item.backup != "" {
				if e := os.Rename(item.backup, item.target); e != nil {
					err = errors.Join(err, e)
					keepBackup = true
				}
			}
		}
		if keepBackup {
			err = errors.Join(err, fmt.Errorf("rollback incomplete; recovery files retained at %s", stage))
		}
	}()
	for i, item := range output {
		if !item.remove {
			if err = os.MkdirAll(filepath.Dir(item.path), 0755); err != nil {
				return err
			}
		}
		current := applied{target: item.path}
		backup := filepath.Join(stage, fmt.Sprintf("old-%d", i))
		if e := os.Rename(item.path, backup); e == nil {
			current.backup = backup
		} else if !errors.Is(e, fs.ErrNotExist) {
			return e
		}
		done = append(done, current)
		if !item.remove {
			if err = os.Rename(filepath.Join(stage, fmt.Sprintf("new-%d", i)), item.path); err != nil {
				return err
			}
			done[len(done)-1].installed = true
		}
	}
	return nil
}
