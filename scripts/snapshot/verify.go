package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{30,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{30,}`),
	regexp.MustCompile(`sk-(?:proj-|ant-)?[A-Za-z0-9_-]{25,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`-----BEGIN (?:RSA |OPENSSH |EC )?PRIVATE KEY-----`),
	regexp.MustCompile(`https?://[^\s/@:]+:[^\s/@]+@`),
}

var bom = []byte("\xef\xbb\xbf")

// verify runs the snapshot's lightweight checks: integrity against PROVENANCE.json, the projects' boundary with GA,
// licences, and known secret patterns. It needs no runtime, network or GPU.
func verify(root string) (string, error) {
	p, err := loadProvenance(root)
	if err != nil {
		return "", err
	}
	var modified []string
	excludedCount := 0
	for _, row := range p.Files {
		path := filepath.Join(root, filepath.FromSlash(row.Path))
		if !row.Included {
			excludedCount++
			if exists(path) {
				return "", fmt.Errorf("excluded upstream data appeared: %s", row.Path)
			}
			continue
		}
		if sum, err := sha256File(path); err != nil {
			return "", err
		} else if sum != row.ExtractedSHA256 {
			return "", fmt.Errorf("%s: SHA-256 %s, PROVENANCE.json records %s", row.Path, sum, row.ExtractedSHA256)
		}
		if row.Modified != nil && *row.Modified {
			modified = append(modified, row.Path)
		}
	}
	slices.Sort(modified)
	if want := editedPaths(); !slices.Equal(modified, want) {
		return "", fmt.Errorf("files marked modified for extraction: %v, expected %v", modified, want)
	}

	csproj, _ := filepath.Glob(filepath.Join(root, "Experiments/Brutal/*/*.csproj"))
	fsproj, _ := filepath.Glob(filepath.Join(root, "Experiments/Brutal/*/*.fsproj"))
	projects := append(csproj, fsproj...)
	if len(projects) != 6 {
		return "", fmt.Errorf("%d projects, expected 6", len(projects))
	}
	lego, err := filepath.Abs(filepath.Join(root, "Experiments/Brutal/Lego/Lego.fsproj"))
	if err != nil {
		return "", err
	}
	for _, project := range projects {
		if err := checkProject(project, lego); err != nil {
			return "", err
		}
	}
	if textures, _ := filepath.Glob(filepath.Join(root, "Experiments/Brutal/Colossus/textures/*.png")); len(textures) != 12 {
		return "", fmt.Errorf("%d textures, expected 12", len(textures))
	}

	skipped := map[string]bool{".git": true, "bin": true, "obj": true, "validation-logs": true, "__pycache__": true}
	files := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		switch {
		case d.IsDir():
			if path != root && skipped[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		case d.Type()&fs.ModeSymlink != 0:
			return fmt.Errorf("symbolic link: %s", rel)
		case !d.Type().IsRegular():
			return nil
		case blocked[strings.ToLower(filepath.Ext(rel))]:
			return fmt.Errorf("blocked file type: %s", rel)
		case strings.HasPrefix(d.Name(), ".env"):
			return fmt.Errorf(".env file: %s", rel)
		}
		files++
		if strings.EqualFold(filepath.Ext(rel), ".png") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := bytes.TrimPrefix(data, bom)
		if !utf8.Valid(text) {
			return fmt.Errorf("not UTF-8 text: %s", rel)
		}
		for _, pattern := range secretPatterns {
			if pattern.Match(text) {
				return fmt.Errorf("secret pattern match in %s; inspect without displaying the match", rel)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	for _, row := range p.LicenceTexts {
		if sum, err := sha256File(filepath.Join(root, filepath.FromSlash(row.Path))); err != nil {
			return "", err
		} else if sum != row.SHA256 {
			return "", fmt.Errorf("%s: SHA-256 %s, PROVENANCE.json records %s", row.Path, sum, row.SHA256)
		}
	}
	for _, c := range [][2]string{
		{"LICENSE", "2021 GuitarAlchemist"},
		{"LICENSES/BRUTAL/LICENSE.txt", "BRUTAL License v1.0"},
		{"NOTICE.txt", "2024 Ahwoo Limited"},
		{"README.md", "gaia/issues/257"},
		{"README.md", "11.0.0-preview.5.26302.115"},
	} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c[0])))
		if err != nil {
			return "", err
		}
		if !bytes.Contains(data, []byte(c[1])) {
			return "", fmt.Errorf("%s no longer says %q", c[0], c[1])
		}
	}
	return fmt.Sprintf("PASS: %d source/licence/document files; 6 projects; 12 textures; all included blobs verified; "+
		"%d excluded assets absent; no GA references or secret-pattern matches.", files, excludedCount), nil
}

// checkProject keeps a project inside the snapshot: its only project reference is the snapshot's own Lego library,
// never GA; it imports nothing; FMOD stays out; its locked restore and the files it copies are present.
func checkProject(project, lego string) error {
	dir := filepath.Dir(project)
	f, err := os.Open(project)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%s: %w", project, err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		attr := func(name string) string {
			for _, a := range el.Attr {
				if a.Name.Local == name {
					return a.Value
				}
			}
			return ""
		}
		include := strings.ReplaceAll(attr("Include"), `\`, "/")
		switch el.Name.Local {
		case "ProjectReference":
			target, err := filepath.Abs(filepath.Join(dir, filepath.FromSlash(include)))
			if err != nil || target != lego {
				return fmt.Errorf("%s references %s", project, include)
			}
		case "Import":
			return fmt.Errorf("%s imports %s", project, attr("Project"))
		case "PackageReference":
			if strings.Contains(strings.ToLower(include), "fmod") {
				return fmt.Errorf("%s references %s", project, include)
			}
		case "None":
			if !strings.ContainsAny(include, "*;") && !exists(filepath.Join(dir, filepath.FromSlash(include))) &&
				!strings.Contains(attr("Condition"), "Exists(") {
				return fmt.Errorf("%s copies %s, which is missing", project, include)
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "packages.lock.json"))
	if err != nil {
		return fmt.Errorf("%s has no packages.lock.json: %w", project, err)
	}
	var lock struct {
		Dependencies map[string]map[string]struct {
			Type string `json:"type"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(data, bom), &lock); err != nil {
		return fmt.Errorf("%s's packages.lock.json: %w", project, err)
	}
	for _, packages := range lock.Dependencies {
		for name, pkg := range packages {
			if strings.EqualFold(pkg.Type, "project") && !strings.EqualFold(name, "lego") {
				return fmt.Errorf("%s's lock has the project %s", project, name)
			}
			if strings.Contains(strings.ToLower(name), "fmod") {
				return fmt.Errorf("%s's lock has %s", project, name)
			}
		}
	}
	return nil
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
