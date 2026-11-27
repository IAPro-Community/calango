package skills

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"calango/internal/core"
)

var (
	// ErrLicenseConfirmation means the skill has no declared or a restrictive license
	// and the request did not set confirm_license.
	ErrLicenseConfirmation = errors.New("skills: importing a skill without a usable declared license needs explicit confirmation")
	// ErrChanged means the skill on disk no longer matches the hash the client saw.
	ErrChanged = errors.New("skills: the skill changed since it was discovered; discover again")
	// ErrReplaceRequired means another version is already imported and the request did
	// not set replace.
	ErrReplaceRequired = errors.New("skills: another version of this skill is imported; set replace to update it")
)

// Import results.
const (
	ResultCreated   = "created"
	ResultUnchanged = "unchanged"
	ResultReplaced  = "replaced"
)

const (
	indexFile    = "index.json"
	storeDir     = "store"
	indexVersion = 1
	versionChars = 24
)

// Imported is one skill copied into the Calango data directory.
type Imported struct {
	// ID equals the SourceID of the directory it was imported from.
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Origin      Origin `json:"origin"`
	SourcePath  string `json:"source_path"`
	Hash        string `json:"hash"`
	License     string `json:"license,omitempty"`
	// LicenseSource is "frontmatter", "file" or "none".
	LicenseSource string `json:"license_source"`
	// LicenseConfirmed records that the person confirmed an import without a usable license.
	LicenseConfirmed bool      `json:"license_confirmed"`
	Warnings         []string  `json:"warnings"`
	Files            int       `json:"files"`
	Bytes            int64     `json:"bytes"`
	ImportedAt       time.Time `json:"imported_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ImportRequest asks to import one discovered skill.
type ImportRequest struct {
	SourceID string
	// Hash is the hash the client saw in discovery; the import fails if it changed.
	Hash string
	// ConfirmLicense accepts a skill whose RequiresLicenseConfirmation is true.
	ConfirmLicense bool
	// Replace allows a new version to replace an imported one.
	Replace bool
}

// AuditFunc records one import event. facts never hold file contents.
type AuditFunc func(ctx context.Context, action, subject string, facts map[string]any) error

// Config configures a Library.
type Config struct {
	// Dir holds the index and the copies, normally <data-dir>/skills.
	Dir string
	// Roots are the only directories discovery reads.
	Roots  []Root
	Limits Limits
	Now    func() time.Time
	// Audit is optional.
	Audit AuditFunc
}

// Library discovers skills in the configured roots and manages imported copies.
// Methods are safe for concurrent use within one process.
type Library struct {
	dir    string
	roots  []Root
	limits Limits
	now    func() time.Time
	audit  AuditFunc
	mu     sync.Mutex
}

type indexEntry struct {
	Imported
	// Version is the directory under store/<id>/ that holds this copy.
	Version string `json:"version"`
}

type indexDoc struct {
	Version int          `json:"version"`
	Skills  []indexEntry `json:"skills"`
}

// New validates cfg, creates the library directory and removes leftovers of
// interrupted imports.
func New(cfg Config) (*Library, error) {
	if cfg.Dir == "" || !filepath.IsAbs(cfg.Dir) {
		return nil, errors.New("skills: an absolute directory is required")
	}
	if cfg.Now == nil {
		return nil, errors.New("skills: clock is required")
	}
	roots := make([]Root, 0, len(cfg.Roots))
	for _, r := range cfg.Roots {
		if _, err := ParseOrigin(string(r.Origin)); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(r.Path) {
			return nil, fmt.Errorf("skills: root %q must be absolute", r.Path)
		}
		clean := filepath.Clean(r.Path)
		if within(clean, filepath.Clean(cfg.Dir)) || within(filepath.Clean(cfg.Dir), clean) {
			return nil, fmt.Errorf("skills: root %q overlaps the library directory", r.Path)
		}
		roots = append(roots, Root{Origin: r.Origin, Path: clean})
	}
	l := &Library{dir: cfg.Dir, roots: roots, limits: cfg.Limits.withDefaults(), now: cfg.Now, audit: cfg.Audit}
	if err := os.MkdirAll(filepath.Join(cfg.Dir, storeDir), 0o700); err != nil {
		return nil, fmt.Errorf("skills: create library: %w", err)
	}
	if err := l.sweep(); err != nil {
		return nil, err
	}
	return l, nil
}

// Roots returns the authorized roots.
func (l *Library) Roots() []Root { return append([]Root(nil), l.roots...) }

// Discover scans the roots and marks each skill new, imported or update_available.
func (l *Library) Discover(ctx context.Context) (Discovery, error) {
	d, err := Discover(ctx, l.roots, l.limits)
	if err != nil {
		return Discovery{}, err
	}
	l.mu.Lock()
	idx, err := l.readIndex()
	l.mu.Unlock()
	if err != nil {
		return Discovery{}, err
	}
	byID := map[string]indexEntry{}
	for _, e := range idx.Skills {
		byID[e.ID] = e
	}
	for i := range d.Skills {
		s := &d.Skills[i]
		switch e, ok := byID[s.SourceID]; {
		case !ok:
			s.ImportStatus = importStatusNew
		case e.Hash == s.Hash:
			s.ImportStatus = importStatusImported
		default:
			s.ImportStatus = importStatusUpdateReady
		}
	}
	return d, nil
}

// List returns the imported skills ordered by name.
func (l *Library) List(context.Context) ([]Imported, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	idx, err := l.readIndex()
	if err != nil {
		return nil, err
	}
	out := make([]Imported, 0, len(idx.Skills))
	for _, e := range idx.Skills {
		out = append(out, e.Imported)
	}
	return out, nil
}

// Dir returns the directory of the active copy of an imported skill, which holds its
// SKILL.md. It is the hook for linking imported skills to agent runs (not wired yet).
func (l *Library) Dir(_ context.Context, id string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	idx, err := l.readIndex()
	if err != nil {
		return "", err
	}
	for _, e := range idx.Skills {
		if e.ID == id {
			return filepath.Join(l.dir, storeDir, e.ID, e.Version), nil
		}
	}
	return "", fmt.Errorf("%w: imported skill %q", core.ErrNotFound, id)
}

// Import copies one discovered skill into the library. Importing the same version again
// is a no-op; a different version needs req.Replace. The source is only read.
func (l *Library) Import(ctx context.Context, req ImportRequest) (Imported, string, error) {
	if req.SourceID == "" || req.Hash == "" {
		return Imported{}, "", fmt.Errorf("%w: source_id and hash are required", core.ErrInvalidInput)
	}
	d, err := Discover(ctx, l.roots, l.limits)
	if err != nil {
		return Imported{}, "", err
	}
	var src *Skill
	for i := range d.Skills {
		if d.Skills[i].SourceID == req.SourceID {
			src = &d.Skills[i]
			break
		}
	}
	if src == nil {
		return Imported{}, "", fmt.Errorf("%w: no discovered skill %q", core.ErrNotFound, req.SourceID)
	}
	if src.Hash != req.Hash {
		return Imported{}, "", ErrChanged
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	idx, err := l.readIndex()
	if err != nil {
		return Imported{}, "", err
	}
	pos := -1
	for i, e := range idx.Skills {
		if e.ID == src.SourceID {
			pos = i
		}
	}
	if pos >= 0 && idx.Skills[pos].Hash == src.Hash {
		return idx.Skills[pos].Imported, ResultUnchanged, nil
	}
	if pos >= 0 && !req.Replace {
		return Imported{}, "", ErrReplaceRequired
	}
	if src.RequiresLicenseConfirmation && !req.ConfirmLicense {
		return Imported{}, "", ErrLicenseConfirmation
	}

	version := strings.TrimPrefix(src.Hash, "sha256:")[:versionChars]
	skillDir := filepath.Join(l.dir, storeDir, src.SourceID)
	if err := l.copySkill(src, skillDir, version); err != nil {
		return Imported{}, "", err
	}
	now := l.now().UTC()
	entry := indexEntry{Version: version, Imported: Imported{
		ID: src.SourceID, Name: src.Name, Description: src.Description, Origin: src.Origin,
		SourcePath: src.Path, Hash: src.Hash, License: src.License, LicenseSource: src.LicenseSource,
		LicenseConfirmed: src.RequiresLicenseConfirmation && req.ConfirmLicense,
		Warnings:         append([]string{}, src.Warnings...), Files: src.Files, Bytes: src.Bytes,
		ImportedAt: now, UpdatedAt: now,
	}}
	result, action := ResultCreated, "skill.import"
	var oldVersion string
	if pos >= 0 {
		entry.ImportedAt = idx.Skills[pos].ImportedAt
		oldVersion = idx.Skills[pos].Version
		idx.Skills[pos] = entry
		result, action = ResultReplaced, "skill.replace"
	} else {
		idx.Skills = append(idx.Skills, entry)
	}
	if err := l.writeIndex(idx); err != nil {
		if version != oldVersion {
			_ = os.RemoveAll(filepath.Join(skillDir, version))
		}
		return Imported{}, "", err
	}
	if oldVersion != "" && oldVersion != version {
		_ = os.RemoveAll(filepath.Join(skillDir, oldVersion))
	}
	l.record(ctx, action, entry.Imported)
	return entry.Imported, result, nil
}

// Remove deletes an imported copy. The source directory is not touched.
func (l *Library) Remove(ctx context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	idx, err := l.readIndex()
	if err != nil {
		return err
	}
	pos := -1
	for i, e := range idx.Skills {
		if e.ID == id {
			pos = i
		}
	}
	if pos < 0 {
		return fmt.Errorf("%w: imported skill %q", core.ErrNotFound, id)
	}
	removed := idx.Skills[pos]
	idx.Skills = append(idx.Skills[:pos], idx.Skills[pos+1:]...)
	if err := l.writeIndex(idx); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(l.dir, storeDir, removed.ID)); err != nil {
		return fmt.Errorf("skills: remove copy: %w", err)
	}
	l.record(ctx, "skill.remove", removed.Imported)
	return nil
}

func (l *Library) record(ctx context.Context, action string, s Imported) {
	if l.audit == nil {
		return
	}
	_ = l.audit(ctx, action, "skill:"+s.ID, map[string]any{
		"name": s.Name, "origin": string(s.Origin), "hash": s.Hash,
		"license": s.License, "license_confirmed": s.LicenseConfirmed,
	})
}

// copySkill writes the skill into skillDir/version through a temporary directory and
// checks that the bytes written hash to the discovered hash.
func (l *Library) copySkill(src *Skill, skillDir, version string) error {
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		return fmt.Errorf("skills: create copy: %w", err)
	}
	tmp, err := os.MkdirTemp(skillDir, ".tmp-")
	if err != nil {
		return fmt.Errorf("skills: create copy: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmp)
		}
	}()
	hash, _, err := hashFiles(src.files, l.limits.MaxSkillBytes, func(f sourceFile, content []byte) error {
		rel := filepath.FromSlash(f.rel)
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("skills: unsafe path %q", f.rel)
		}
		dst := filepath.Join(tmp, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if f.exec {
			mode = 0o700
		}
		return writeNew(dst, content, mode)
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrChanged
		}
		return fmt.Errorf("skills: copy: %w", err)
	}
	if hash != src.Hash {
		return ErrChanged
	}
	final := filepath.Join(skillDir, version)
	if err := os.RemoveAll(final); err != nil {
		return fmt.Errorf("skills: replace stale copy: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		return fmt.Errorf("skills: activate copy: %w", err)
	}
	ok = true
	return nil
}

// hexID reports whether s is n lowercase hex digits, the only shape ids and versions
// take on disk.
func hexID(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func writeNew(path string, content []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (l *Library) readIndex() (indexDoc, error) {
	b, err := os.ReadFile(filepath.Join(l.dir, indexFile))
	if errors.Is(err, os.ErrNotExist) {
		return indexDoc{Version: indexVersion, Skills: []indexEntry{}}, nil
	}
	if err != nil {
		return indexDoc{}, fmt.Errorf("skills: read index: %w", err)
	}
	var idx indexDoc
	if err := json.Unmarshal(b, &idx); err != nil {
		return indexDoc{}, fmt.Errorf("skills: decode index: %w", err)
	}
	if idx.Version != indexVersion {
		return indexDoc{}, fmt.Errorf("skills: index version %d is not supported", idx.Version)
	}
	if idx.Skills == nil {
		idx.Skills = []indexEntry{}
	}
	for _, e := range idx.Skills {
		if !hexID(e.ID, sourceIDHexLength) || !hexID(e.Version, versionChars) {
			return indexDoc{}, fmt.Errorf("skills: index holds a malformed entry")
		}
	}
	return idx, nil
}

// writeIndex replaces index.json atomically: write a temporary file, sync, rename.
func (l *Library) writeIndex(idx indexDoc) error {
	idx.Version = indexVersion
	sort.SliceStable(idx.Skills, func(i, j int) bool {
		if idx.Skills[i].Name != idx.Skills[j].Name {
			return idx.Skills[i].Name < idx.Skills[j].Name
		}
		return idx.Skills[i].ID < idx.Skills[j].ID
	})
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tmp := filepath.Join(l.dir, indexFile+"."+hex.EncodeToString(suffix[:])+".tmp")
	if err := writeNew(tmp, append(b, '\n'), 0o600); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("skills: write index: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(l.dir, indexFile)); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("skills: write index: %w", err)
	}
	if d, err := os.Open(l.dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// sweep removes temporary files and copies the index does not reference. It only
// touches the library directory.
func (l *Library) sweep() error {
	idx, err := l.readIndex()
	if err != nil {
		return err
	}
	keep := map[string]string{}
	for _, e := range idx.Skills {
		keep[e.ID] = e.Version
	}
	if tmps, err := filepath.Glob(filepath.Join(l.dir, indexFile+".*.tmp")); err == nil {
		for _, t := range tmps {
			_ = os.Remove(t)
		}
	}
	store := filepath.Join(l.dir, storeDir)
	ids, err := os.ReadDir(store)
	if err != nil {
		return fmt.Errorf("skills: read library: %w", err)
	}
	for _, id := range ids {
		version, ok := keep[id.Name()]
		if !ok {
			_ = os.RemoveAll(filepath.Join(store, id.Name()))
			continue
		}
		versions, err := os.ReadDir(filepath.Join(store, id.Name()))
		if err != nil {
			continue
		}
		for _, v := range versions {
			if v.Name() != version {
				_ = os.RemoveAll(filepath.Join(store, id.Name(), v.Name()))
			}
		}
	}
	return nil
}
