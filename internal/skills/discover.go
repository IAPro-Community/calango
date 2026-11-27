// Package skills discovers Claude Code and Codex skills in authorized directories and
// imports versioned copies into the Calango data directory.
//
// Discovery is read-only: it opens files for reading and never writes, renames or
// changes the mode of anything under a source root. Only the roots in Config are
// scanned, one level deep; symlinks are followed only when they resolve inside the
// root they were found in. Imported skills are not wired into agent runs yet; see
// Library.Dir and DEV/SPECS/modules/skills.md.
package skills

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Origin is where a skill was found.
type Origin string

const (
	// OriginClaude is a Claude Code personal skills directory (~/.claude/skills).
	OriginClaude Origin = "claude"
	// OriginCodex is a Codex skills directory ($CODEX_HOME/skills, default ~/.codex/skills).
	OriginCodex Origin = "codex"
	// OriginProject is a project skills directory (<project>/.claude/skills or similar).
	OriginProject Origin = "project"
)

// ParseOrigin accepts the three known origins.
func ParseOrigin(s string) (Origin, error) {
	switch Origin(s) {
	case OriginClaude, OriginCodex, OriginProject:
		return Origin(s), nil
	}
	return "", fmt.Errorf("skills: unknown origin %q (want claude, codex or project)", s)
}

// Root is one authorized directory whose subdirectories are skills.
type Root struct {
	Origin Origin `json:"origin"`
	Path   string `json:"path"`
}

// Limits bound one discovery. Zero fields take the DefaultLimits value.
type Limits struct {
	// MaxSkills caps the skills returned across all roots.
	MaxSkills int
	// MaxEntriesPerRoot caps the directory entries read from one root.
	MaxEntriesPerRoot int
	// MaxFilesPerSkill caps the files of one skill directory.
	MaxFilesPerSkill int
	// MaxDepth caps the directory nesting inside one skill.
	MaxDepth int
	// MaxSkillMDBytes caps SKILL.md.
	MaxSkillMDBytes int64
	// MaxSkillBytes caps the total size of one skill directory.
	MaxSkillBytes int64
}

// DefaultLimits are the limits used when Config.Limits leaves a field at zero.
func DefaultLimits() Limits {
	return Limits{
		MaxSkills:         256,
		MaxEntriesPerRoot: 1024,
		MaxFilesPerSkill:  128,
		MaxDepth:          6,
		MaxSkillMDBytes:   256 << 10,
		MaxSkillBytes:     8 << 20,
	}
}

func (l Limits) withDefaults() Limits {
	d := DefaultLimits()
	if l.MaxSkills <= 0 {
		l.MaxSkills = d.MaxSkills
	}
	if l.MaxEntriesPerRoot <= 0 {
		l.MaxEntriesPerRoot = d.MaxEntriesPerRoot
	}
	if l.MaxFilesPerSkill <= 0 {
		l.MaxFilesPerSkill = d.MaxFilesPerSkill
	}
	if l.MaxDepth <= 0 {
		l.MaxDepth = d.MaxDepth
	}
	if l.MaxSkillMDBytes <= 0 {
		l.MaxSkillMDBytes = d.MaxSkillMDBytes
	}
	if l.MaxSkillBytes <= 0 {
		l.MaxSkillBytes = d.MaxSkillBytes
	}
	return l
}

// Warning codes. They are stable: clients translate them.
const (
	WarnNoLicense           = "no_license"
	WarnLicenseRestrictive  = "license_restrictive"
	WarnMissingFrontmatter  = "missing_frontmatter"
	WarnInvalidFrontmatter  = "invalid_frontmatter"
	WarnMissingName         = "missing_name"
	WarnInvalidName         = "invalid_name"
	WarnMissingDescription  = "missing_description"
	WarnDescriptionTooLong  = "description_truncated"
	WarnSymlinkSkipped      = "symlink_skipped"
	WarnSpecialFileSkipped  = "special_file_skipped"
	maxDescriptionRunes     = 1024
	maxLicenseRunes         = 200
	licenseSniffBytes       = 4 << 10
	sourceIDHexLength       = 32
	importStatusNew         = "new"
	importStatusImported    = "imported"
	importStatusUpdateReady = "update_available"
)

// Skip reasons. They are stable: clients translate them.
const (
	SkipSymlinkOutsideRoot = "symlink_outside_root"
	SkipBrokenSymlink      = "broken_symlink"
	SkipNoSkillMD          = "no_skill_md"
	SkipSkillMDTooLarge    = "skill_md_too_large"
	SkipTooManyFiles       = "too_many_files"
	SkipTooLarge           = "too_large"
	SkipTooDeep            = "too_deep"
	SkipUnreadable         = "unreadable"
)

// Root states reported by Discover.
const (
	RootOK           = "ok"
	RootMissing      = "missing"
	RootNotDirectory = "not_directory"
	RootUnreadable   = "unreadable"
)

// Skill is one skill found under a root.
type Skill struct {
	// SourceID identifies the skill directory: a hash of origin, root and entry name.
	// It is stable across scans and is the key of an imported copy.
	SourceID    string `json:"source_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Origin      Origin `json:"origin"`
	Root        string `json:"root"`
	// Path is the skill directory as found under the root (not symlink-resolved).
	Path string `json:"path"`
	// Hash is "sha256:<hex>" over the sorted file list, sizes and contents.
	Hash string `json:"hash"`
	// License is the declared license: the frontmatter "license" field or what a
	// LICENSE/COPYING file next to SKILL.md says. Empty when none is declared.
	License string `json:"license,omitempty"`
	// LicenseSource is "frontmatter", "file" or "none".
	LicenseSource string `json:"license_source"`
	// RequiresLicenseConfirmation is true without a declared license or with a
	// restrictive one; importing then needs confirm_license.
	RequiresLicenseConfirmation bool     `json:"requires_license_confirmation"`
	Warnings                    []string `json:"warnings"`
	Files                       int      `json:"files"`
	Bytes                       int64    `json:"bytes"`
	// ImportStatus is "new", "imported" (same hash) or "update_available".
	ImportStatus string `json:"import_status"`

	files []sourceFile
}

// Skipped is a directory that looked like a skill but was not read.
type Skipped struct {
	Origin Origin `json:"origin"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// RootStatus reports one configured root.
type RootStatus struct {
	Origin Origin `json:"origin"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

// Discovery is the result of one scan.
type Discovery struct {
	Roots   []RootStatus `json:"roots"`
	Skills  []Skill      `json:"skills"`
	Skipped []Skipped    `json:"skipped"`
	// Truncated is true when a limit stopped the scan before every entry was read.
	Truncated bool `json:"truncated"`
}

type sourceFile struct {
	rel  string // slash-separated path inside the skill
	abs  string // resolved path to read
	size int64
	exec bool
}

// SourceID returns the stable id of the skill directory entry under root.
func SourceID(root Root, entry string) string {
	sum := sha256.Sum256([]byte(string(root.Origin) + "\x00" + filepath.Clean(root.Path) + "\x00" + entry))
	return hex.EncodeToString(sum[:])[:sourceIDHexLength]
}

// Discover scans roots read-only. A root that is missing or unreadable is reported in
// Roots and does not fail the scan. Only ctx cancellation returns an error.
func Discover(ctx context.Context, roots []Root, limits Limits) (Discovery, error) {
	limits = limits.withDefaults()
	out := Discovery{Roots: []RootStatus{}, Skills: []Skill{}, Skipped: []Skipped{}}
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return Discovery{}, err
		}
		status := scanRoot(ctx, root, limits, &out)
		out.Roots = append(out.Roots, RootStatus{Origin: root.Origin, Path: root.Path, Status: status})
	}
	if err := ctx.Err(); err != nil {
		return Discovery{}, err
	}
	return out, nil
}

func scanRoot(ctx context.Context, root Root, limits Limits, out *Discovery) string {
	resolved, err := filepath.EvalSymlinks(root.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return RootMissing
		}
		return RootUnreadable
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return RootUnreadable
	}
	if !info.IsDir() {
		return RootNotDirectory
	}
	dir, err := os.Open(resolved)
	if err != nil {
		return RootUnreadable
	}
	entries, err := dir.ReadDir(limits.MaxEntriesPerRoot + 1)
	_ = dir.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return RootUnreadable
	}
	if len(entries) > limits.MaxEntriesPerRoot {
		entries = entries[:limits.MaxEntriesPerRoot]
		out.Truncated = true
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if ctx.Err() != nil {
			return RootOK
		}
		name := entry.Name()
		// Hidden entries (for example Codex's bundled .system skills) are not the person's.
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(root.Path, name)
		skip := func(reason string) {
			out.Skipped = append(out.Skipped, Skipped{Origin: root.Origin, Path: path, Reason: reason})
		}
		dirPath := filepath.Join(resolved, name)
		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			target, err := filepath.EvalSymlinks(dirPath)
			if err != nil {
				skip(SkipBrokenSymlink)
				continue
			}
			if !within(resolved, target) {
				skip(SkipSymlinkOutsideRoot)
				continue
			}
			ti, err := os.Stat(target)
			if err != nil || !ti.IsDir() {
				continue
			}
			dirPath = target
		case entry.IsDir():
		default:
			continue
		}
		if len(out.Skills) >= limits.MaxSkills {
			out.Truncated = true
			return RootOK
		}
		skill, reason := readSkill(ctx, root, resolved, name, dirPath, limits)
		if reason != "" {
			skip(reason)
			continue
		}
		skill.Path = path
		out.Skills = append(out.Skills, skill)
	}
	return RootOK
}

// within reports whether target is base or below it. Both must be resolved paths.
func within(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

type walkState struct {
	root     string
	limits   Limits
	files    []sourceFile
	bytes    int64
	warnings map[string]bool
}

// readSkill reads one skill directory. A non-empty reason means it was skipped.
func readSkill(ctx context.Context, root Root, resolvedRoot, entry, dir string, limits Limits) (Skill, string) {
	mdInfo, err := os.Lstat(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Skill{}, SkipNoSkillMD
		}
		return Skill{}, SkipUnreadable
	}
	if mdInfo.Mode()&fs.ModeSymlink == 0 && !mdInfo.Mode().IsRegular() {
		return Skill{}, SkipNoSkillMD
	}
	st := &walkState{root: resolvedRoot, limits: limits, warnings: map[string]bool{}}
	if reason := st.walk(ctx, dir, "", 0); reason != "" {
		return Skill{}, reason
	}
	sort.Slice(st.files, func(i, j int) bool { return st.files[i].rel < st.files[j].rel })
	var md *sourceFile
	for i := range st.files {
		if st.files[i].rel == "SKILL.md" {
			md = &st.files[i]
		}
	}
	if md == nil {
		// SKILL.md is a symlink that resolved outside the root or to a non-file.
		return Skill{}, SkipNoSkillMD
	}
	if md.size > limits.MaxSkillMDBytes {
		return Skill{}, SkipSkillMDTooLarge
	}
	hash, mdBytes, err := hashFiles(st.files, limits.MaxSkillBytes, nil)
	if err != nil {
		return Skill{}, SkipUnreadable
	}
	skill := Skill{
		SourceID: SourceID(root, entry),
		Origin:   root.Origin,
		Root:     root.Path,
		Hash:     hash,
		Files:    len(st.files),
		Bytes:    st.bytes,
		files:    st.files,
	}
	fields, ferr := parseFrontmatter(mdBytes)
	switch {
	case errors.Is(ferr, errNoFrontmatter):
		st.warnings[WarnMissingFrontmatter] = true
	case ferr != nil:
		st.warnings[WarnInvalidFrontmatter] = true
	}
	skill.Name = strings.TrimSpace(fields["name"])
	switch {
	case skill.Name == "":
		skill.Name = entry
		st.warnings[WarnMissingName] = true
	case !validName(skill.Name):
		st.warnings[WarnInvalidName] = true
		skill.Name = truncateRunes(skill.Name, 64)
	}
	skill.Description = strings.TrimSpace(fields["description"])
	if skill.Description == "" {
		st.warnings[WarnMissingDescription] = true
	} else if d := truncateRunes(skill.Description, maxDescriptionRunes); d != skill.Description {
		skill.Description = d
		st.warnings[WarnDescriptionTooLong] = true
	}
	skill.License, skill.LicenseSource = declaredLicense(fields["license"], st.files)
	if skill.License == "" {
		st.warnings[WarnNoLicense] = true
	} else if restrictive(skill.License) {
		st.warnings[WarnLicenseRestrictive] = true
	}
	skill.RequiresLicenseConfirmation = st.warnings[WarnNoLicense] || st.warnings[WarnLicenseRestrictive]
	skill.Warnings = make([]string, 0, len(st.warnings))
	for w := range st.warnings {
		skill.Warnings = append(skill.Warnings, w)
	}
	sort.Strings(skill.Warnings)
	return skill, ""
}

func (st *walkState) walk(ctx context.Context, dir, rel string, depth int) string {
	if depth > st.limits.MaxDepth {
		return SkipTooDeep
	}
	if ctx.Err() != nil {
		return SkipUnreadable
	}
	f, err := os.Open(dir)
	if err != nil {
		return SkipUnreadable
	}
	entries, err := f.ReadDir(st.limits.MaxFilesPerSkill + 1)
	_ = f.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return SkipUnreadable
	}
	if len(entries) > st.limits.MaxFilesPerSkill {
		return SkipTooManyFiles
	}
	for _, e := range entries {
		name := e.Name()
		if name == ".git" || name == ".DS_Store" {
			continue
		}
		path := filepath.Join(dir, name)
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		mode := e.Type()
		target := path
		if mode&fs.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil || !within(st.root, resolved) {
				st.warnings[WarnSymlinkSkipped] = true
				continue
			}
			info, err := os.Stat(resolved)
			if err != nil || !info.Mode().IsRegular() {
				// Directory symlinks are not followed, to rule out cycles.
				st.warnings[WarnSymlinkSkipped] = true
				continue
			}
			target, mode = resolved, 0
		}
		switch {
		case mode.IsDir():
			if reason := st.walk(ctx, path, childRel, depth+1); reason != "" {
				return reason
			}
		case mode.IsRegular():
			info, err := os.Stat(target)
			if err != nil {
				return SkipUnreadable
			}
			if len(st.files) >= st.limits.MaxFilesPerSkill {
				return SkipTooManyFiles
			}
			st.bytes += info.Size()
			if st.bytes > st.limits.MaxSkillBytes {
				return SkipTooLarge
			}
			st.files = append(st.files, sourceFile{rel: childRel, abs: target, size: info.Size(), exec: info.Mode()&0o111 != 0})
		default:
			st.warnings[WarnSpecialFileSkipped] = true
		}
	}
	return ""
}

// hashFiles reads every file once, checks its size against the listing and returns the
// skill hash and the SKILL.md bytes. When sink is set it receives each file's content.
func hashFiles(files []sourceFile, maxBytes int64, sink func(sourceFile, []byte) error) (string, []byte, error) {
	h := sha256.New()
	var md []byte
	var total int64
	for _, f := range files {
		content, err := readLimited(f.abs, f.size)
		if err != nil {
			return "", nil, err
		}
		total += int64(len(content))
		if total > maxBytes {
			return "", nil, fmt.Errorf("skills: %s grew past the size limit", f.rel)
		}
		sum := sha256.Sum256(content)
		fmt.Fprintf(h, "%s\x00%d\x00%x\n", f.rel, len(content), sum)
		if f.rel == "SKILL.md" {
			md = content
		}
		if sink != nil {
			if err := sink(f, content); err != nil {
				return "", nil, err
			}
		}
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), md, nil
}

// readLimited reads a file that the listing says has size bytes. A file that grew is an
// error, so a concurrent writer cannot push a skill past its limits.
func readLimited(path string, size int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, size+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > size {
		return nil, fmt.Errorf("skills: %s changed while reading", path)
	}
	return content, nil
}

var skillName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func validName(name string) bool { return skillName.MatchString(name) }

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

var licenseFiles = map[string]bool{
	"license": true, "license.txt": true, "license.md": true,
	"copying": true, "copying.txt": true, "copying.md": true,
}

// declaredLicense returns the frontmatter license or, failing that, what a license file
// at the top of the skill says.
func declaredLicense(front string, files []sourceFile) (string, string) {
	if l := strings.TrimSpace(front); l != "" {
		return truncateRunes(l, maxLicenseRunes), "frontmatter"
	}
	for _, f := range files {
		if strings.Contains(f.rel, "/") || !licenseFiles[strings.ToLower(f.rel)] {
			continue
		}
		head, err := readHead(f.abs, licenseSniffBytes)
		if err != nil || len(bytes.TrimSpace(head)) == 0 {
			continue
		}
		return identifyLicense(head, f.rel), "file"
	}
	return "", "none"
}

func readHead(path string, n int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, n))
}

// identifyLicense names common licenses from the head of a license file. Anything
// unrecognised is reported by its first line, so the person can judge it.
func identifyLicense(head []byte, file string) string {
	text := strings.ToLower(string(head))
	known := []struct{ needle, id string }{
		{"gnu affero general public license", "AGPL-3.0"},
		{"gnu lesser general public license", "LGPL"},
		{"gnu general public license", "GPL"},
		{"apache license", "Apache-2.0"},
		{"mozilla public license", "MPL-2.0"},
		{"mit license", "MIT"},
		{"permission is hereby granted, free of charge", "MIT"},
		{"bsd 3-clause", "BSD-3-Clause"},
		{"bsd 2-clause", "BSD-2-Clause"},
		{"redistribution and use in source and binary forms", "BSD"},
		{"creative commons", "CC"},
		{"this is free and unencumbered software released into the public domain", "Unlicense"},
		{"isc license", "ISC"},
	}
	for _, k := range known {
		if strings.Contains(text, k.needle) {
			return k.id + " (" + file + ")"
		}
	}
	for _, line := range strings.Split(string(head), "\n") {
		if l := strings.TrimSpace(line); l != "" {
			return truncateRunes(l, maxLicenseRunes-len(file)-3) + " (" + file + ")"
		}
	}
	return file
}

// restrictive flags licenses that do not grant reuse, so importing needs confirmation.
func restrictive(license string) bool {
	l := strings.ToLower(license)
	for _, needle := range []string{"proprietary", "all rights reserved", "unlicensed", "no license", "confidential"} {
		if strings.Contains(l, needle) {
			return true
		}
	}
	return false
}
