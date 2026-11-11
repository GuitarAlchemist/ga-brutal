package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// provenance is PROVENANCE.json field for field, in its order, so that a sync rewrites only what changed. A field this
// type does not know fails the load instead of being dropped on the next write.
type provenance struct {
	SourceRepository      string        `json:"source_repository"`
	SourcePullRequest     string        `json:"source_pull_request"`
	SourceCommit          string        `json:"source_commit"`
	SourceSubtree         string        `json:"source_subtree"`
	SourceTree            string        `json:"source_tree"`
	CoordinationEpic      string        `json:"coordination_epic"`
	BrutalLicenceRevision string        `json:"brutal_licence_revision"`
	OwnerSession          string        `json:"owner_session"`
	OwnerAcknowledgement  string        `json:"owner_acknowledgement"`
	SnapshotMethod        string        `json:"snapshot_method"`
	Files                 []fileRow     `json:"files"`
	LicenceTexts          []licenceText `json:"licence_texts"`
	ExtractionChanges     []string      `json:"extraction_changes"`
	SourcePullRequestBase string        `json:"source_pull_request_base"`
	PreviousSourceCommit  string        `json:"previous_source_commit"`
}

// fileRow is one file of the GA subtree: its upstream blob, and either the bytes extracted from it or why it was left out.
type fileRow struct {
	Path            string `json:"path"`
	GitBlob         string `json:"git_blob"`
	SHA256          string `json:"sha256"`
	Bytes           int64  `json:"bytes"`
	Included        bool   `json:"included"`
	ExtractedSHA256 string `json:"extracted_sha256,omitempty"`
	Modified        *bool  `json:"modified_for_extraction,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

type licenceText struct {
	Path    string `json:"path"`
	Package string `json:"package"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

func loadProvenance(root string) (*provenance, error) {
	data, err := os.ReadFile(filepath.Join(root, "PROVENANCE.json"))
	if err != nil {
		return nil, fmt.Errorf("%w (run from the repository root)", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p provenance
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("PROVENANCE.json: %w", err)
	}
	return &p, nil
}

// encode writes the format of the first extraction: two-space indent, UTF-8 left as it is, a final newline.
func (p *provenance) encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(p)
	return buf.Bytes(), err
}
