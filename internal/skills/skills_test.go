package skills

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skillMD(name, desc, license string) string {
	b := "---\nname: " + name + "\ndescription: " + desc + "\n"
	if license != "" {
		b += "license: " + license + "\n"
	}
	return b + "---\n\n# " + name + "\n\nSteps.\n"
}

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlinks unavailable: %v", err)
		}
		t.Fatal(err)
	}
}

func findSkill(t *testing.T, d Discovery, name string) Skill {
	t.Helper()
	for _, s := range d.Skills {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("skill %q not discovered; got %+v skipped %+v", name, d.Skills, d.Skipped)
	return Skill{}
}

func skippedReason(d Discovery, base string) string {
	for _, s := range d.Skipped {
		if filepath.Base(s.Path) == base {
			return s.Reason
		}
	}
	return ""
}

func TestFrontmatterSubset(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]string
		err  error
	}{
		{"plain", "---\nname: pdf\ndescription: Read PDFs # comment\n---\nbody", map[string]string{"name": "pdf", "description": "Read PDFs"}, nil},
		{"quoted", "---\nname: \"a\\tb\"\nlicense: 'it''s MIT'\n---\n", map[string]string{"name": "a\tb", "license": "it's MIT"}, nil},
		{"literal block", "---\ndescription: |\n  line one\n  line two\nname: x\n---\n", map[string]string{"description": "line one\nline two\n", "name": "x"}, nil},
		{"folded strip", "---\ndescription: >-\n  one\n  two\n---\n", map[string]string{"description": "one two"}, nil},
		{"nested skipped", "---\nname: x\nmetadata:\n  version: 1\nallowed-tools:\n- Read\n---\n", map[string]string{"name": "x", "metadata": "", "allowed-tools": ""}, nil},
		{"crlf and bom", "\xef\xbb\xbf---\r\nname: x\r\n---\r\n", map[string]string{"name": "x"}, nil},
		{"no frontmatter", "# Title\n", nil, errNoFrontmatter},
		{"unterminated", "---\nname: x\n", nil, errFrontmatter},
		{"tab indent", "---\nname: x\n\tbad: y\n---\n", nil, errFrontmatter},
		{"duplicate", "---\nname: x\nname: y\n---\n", nil, errFrontmatter},
		{"no colon", "---\njust text\n---\n", nil, errFrontmatter},
		{"alias", "---\nname: *ref\n---\n", nil, errFrontmatter},
		{"bad quote", "---\nname: \"open\n---\n", nil, errFrontmatter},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFrontmatter([]byte(tc.in))
			if !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			if tc.err == nil && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("fields = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestDiscoverReportsOriginLicenseAndWarnings(t *testing.T) {
	claude := filepath.Join(t.TempDir(), "claude-skills")
	codex := filepath.Join(t.TempDir(), "codex-skills")
	writeFile(t, filepath.Join(claude, "pdf", "SKILL.md"), skillMD("pdf", "Fill PDF forms", "MIT"))
	writeFile(t, filepath.Join(claude, "pdf", "scripts", "fill.py"), "print('x')\n")
	writeFile(t, filepath.Join(claude, "bare", "SKILL.md"), skillMD("bare", "No license here", ""))
	writeFile(t, filepath.Join(claude, "broken", "SKILL.md"), "---\nname: broken\n\tdescription: tab\n---\n")
	writeFile(t, filepath.Join(claude, "notes", "README.md"), "not a skill")
	writeFile(t, filepath.Join(claude, "loose.md"), "a file at the root is ignored")
	writeFile(t, filepath.Join(codex, "deploy", "SKILL.md"), skillMD("deploy", "Deploy", ""))
	writeFile(t, filepath.Join(codex, "deploy", "LICENSE"), "Apache License\nVersion 2.0\n")
	writeFile(t, filepath.Join(codex, "secret", "SKILL.md"), skillMD("secret", "Internal", "Proprietary. All rights reserved"))
	writeFile(t, filepath.Join(codex, ".system", "builtin", "SKILL.md"), skillMD("builtin", "Bundled", "MIT"))
	missing := filepath.Join(t.TempDir(), "absent")

	d, err := Discover(context.Background(), []Root{
		{Origin: OriginClaude, Path: claude}, {Origin: OriginCodex, Path: codex}, {Origin: OriginProject, Path: missing},
	}, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{d.Roots[0].Status, d.Roots[1].Status, d.Roots[2].Status}; !slices.Equal(got, []string{RootOK, RootOK, RootMissing}) {
		t.Fatalf("root states = %v", got)
	}
	pdf := findSkill(t, d, "pdf")
	if pdf.Origin != OriginClaude || pdf.License != "MIT" || pdf.LicenseSource != "frontmatter" || pdf.RequiresLicenseConfirmation {
		t.Fatalf("pdf = %+v", pdf)
	}
	if pdf.Files != 2 || !strings.HasPrefix(pdf.Hash, "sha256:") || pdf.Path != filepath.Join(claude, "pdf") || len(pdf.Warnings) != 0 {
		t.Fatalf("pdf listing = %+v", pdf)
	}
	bare := findSkill(t, d, "bare")
	if !bare.RequiresLicenseConfirmation || !slices.Contains(bare.Warnings, WarnNoLicense) || bare.LicenseSource != "none" {
		t.Fatalf("bare = %+v", bare)
	}
	broken := findSkill(t, d, "broken")
	if !slices.Contains(broken.Warnings, WarnInvalidFrontmatter) || !slices.Contains(broken.Warnings, WarnMissingName) {
		t.Fatalf("broken = %+v", broken)
	}
	deploy := findSkill(t, d, "deploy")
	if deploy.Origin != OriginCodex || deploy.License != "Apache-2.0 (LICENSE)" || deploy.LicenseSource != "file" || deploy.RequiresLicenseConfirmation {
		t.Fatalf("deploy = %+v", deploy)
	}
	secret := findSkill(t, d, "secret")
	if !secret.RequiresLicenseConfirmation || !slices.Contains(secret.Warnings, WarnLicenseRestrictive) {
		t.Fatalf("secret = %+v", secret)
	}
	if skippedReason(d, "notes") != SkipNoSkillMD {
		t.Fatalf("notes skipped = %+v", d.Skipped)
	}
	for _, s := range d.Skills {
		if s.Name == "builtin" {
			t.Fatal("hidden .system directory must not be scanned")
		}
	}
	if len(d.Skills) != 5 || d.Truncated {
		t.Fatalf("skills = %d truncated = %v", len(d.Skills), d.Truncated)
	}
	again, err := Discover(context.Background(), []Root{{Origin: OriginClaude, Path: claude}}, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if p := findSkill(t, again, "pdf"); p.Hash != pdf.Hash || p.SourceID != pdf.SourceID {
		t.Fatal("hash and source id must be stable across scans")
	}
}

func TestDiscoverDoesNotFollowSymlinksOutOfRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "evil", "SKILL.md"), skillMD("evil", "Outside", "MIT"))
	writeFile(t, filepath.Join(outside, "passwd"), "secret")
	writeFile(t, filepath.Join(root, "real", "SKILL.md"), skillMD("real", "Inside", "MIT"))
	writeFile(t, filepath.Join(root, "leaky", "SKILL.md"), skillMD("leaky", "Links out", "MIT"))
	symlinkOrSkip(t, filepath.Join(outside, "evil"), filepath.Join(root, "evil"))
	symlinkOrSkip(t, filepath.Join(root, "real"), filepath.Join(root, "alias"))
	symlinkOrSkip(t, filepath.Join(outside, "passwd"), filepath.Join(root, "leaky", "passwd"))
	symlinkOrSkip(t, filepath.Join(root, "nowhere"), filepath.Join(root, "dangling"))

	d, err := Discover(context.Background(), []Root{{Origin: OriginProject, Path: root}}, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if got := skippedReason(d, "evil"); got != SkipSymlinkOutsideRoot {
		t.Fatalf("evil reason = %q", got)
	}
	if got := skippedReason(d, "dangling"); got != SkipBrokenSymlink {
		t.Fatalf("dangling reason = %q", got)
	}
	leaky := findSkill(t, d, "leaky")
	if leaky.Files != 1 || !slices.Contains(leaky.Warnings, WarnSymlinkSkipped) {
		t.Fatalf("leaky = %+v", leaky)
	}
	var aliases int
	for _, s := range d.Skills {
		if s.Name == "real" {
			aliases++
		}
	}
	if aliases != 2 {
		t.Fatalf("a symlink inside the root should be read; got %d copies of real", aliases)
	}
}

func TestDiscoverEnforcesLimits(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"a", "b", "c"} {
		writeFile(t, filepath.Join(root, n, "SKILL.md"), skillMD(n, "x", "MIT"))
	}
	d, err := Discover(context.Background(), []Root{{Origin: OriginClaude, Path: root}}, Limits{MaxSkills: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Skills) != 2 || !d.Truncated {
		t.Fatalf("max skills: got %d truncated=%v", len(d.Skills), d.Truncated)
	}

	root = t.TempDir()
	writeFile(t, filepath.Join(root, "many", "SKILL.md"), skillMD("many", "x", "MIT"))
	for i := 0; i < 4; i++ {
		writeFile(t, filepath.Join(root, "many", "f"+string(rune('0'+i))), "x")
	}
	writeFile(t, filepath.Join(root, "big", "SKILL.md"), skillMD("big", "x", "MIT"))
	writeFile(t, filepath.Join(root, "big", "blob"), strings.Repeat("x", 2048))
	writeFile(t, filepath.Join(root, "hugemd", "SKILL.md"), skillMD("hugemd", strings.Repeat("y", 900), "MIT"))
	writeFile(t, filepath.Join(root, "deep", "SKILL.md"), skillMD("deep", "x", "MIT"))
	writeFile(t, filepath.Join(root, "deep", "a", "b", "c", "d"), "x")
	d, err = Discover(context.Background(), []Root{{Origin: OriginClaude, Path: root}}, Limits{
		MaxFilesPerSkill: 4, MaxSkillBytes: 1500, MaxSkillMDBytes: 800, MaxDepth: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"many": SkipTooManyFiles, "big": SkipTooLarge, "hugemd": SkipSkillMDTooLarge, "deep": SkipTooDeep}
	for name, reason := range want {
		if got := skippedReason(d, name); got != reason {
			t.Errorf("%s: reason %q, want %q (skipped %+v)", name, got, reason, d.Skipped)
		}
	}
	if len(d.Skills) != 0 {
		t.Fatalf("no skill should pass the limits: %+v", d.Skills)
	}
}

func TestDiscoverHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Discover(ctx, []Root{{Origin: OriginClaude, Path: t.TempDir()}}, Limits{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

type snapshot map[string]string

// snap records every path under dir with its mode, mtime and content.
func snap(t *testing.T, dir string) snapshot {
	t.Helper()
	out := snapshot{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		entry := info.Mode().String() + " " + info.ModTime().Format(time.RFC3339Nano)
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry += " " + string(b)
		}
		out[path] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type auditCall struct{ action, subject string }

func newLibrary(t *testing.T, roots ...Root) (*Library, string, *[]auditCall) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "skills")
	var calls []auditCall
	lib, err := New(Config{
		Dir: dir, Roots: roots,
		Now: func() time.Time { return time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC) },
		Audit: func(_ context.Context, action, subject string, _ map[string]any) error {
			calls = append(calls, auditCall{action, subject})
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return lib, dir, &calls
}

func discovered(t *testing.T, lib *Library, name string) Skill {
	t.Helper()
	d, err := lib.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return findSkill(t, d, name)
}

func versionDirs(t *testing.T, dir, id string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, storeDir, id))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestImportIsIdempotentAndLeavesSourceIntact(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pdf", "SKILL.md"), skillMD("pdf", "Fill PDF forms", "MIT"))
	writeFile(t, filepath.Join(root, "pdf", "scripts", "fill.sh"), "#!/bin/sh\necho hi\n")
	if err := os.Chmod(filepath.Join(root, "pdf", "scripts", "fill.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := snap(t, root)
	lib, dir, calls := newLibrary(t, Root{Origin: OriginClaude, Path: root})
	ctx := context.Background()

	pdf := discovered(t, lib, "pdf")
	if pdf.ImportStatus != importStatusNew {
		t.Fatalf("status = %q", pdf.ImportStatus)
	}
	got, result, err := lib.Import(ctx, ImportRequest{SourceID: pdf.SourceID, Hash: pdf.Hash})
	if err != nil || result != ResultCreated {
		t.Fatalf("import: %v %q", err, result)
	}
	if got.Name != "pdf" || got.Origin != OriginClaude || got.Hash != pdf.Hash || got.SourcePath != pdf.Path || got.LicenseConfirmed {
		t.Fatalf("imported = %+v", got)
	}
	again, result, err := lib.Import(ctx, ImportRequest{SourceID: pdf.SourceID, Hash: pdf.Hash})
	if err != nil || result != ResultUnchanged || again.ImportedAt != got.ImportedAt {
		t.Fatalf("reimport: %v %q", err, result)
	}
	list, err := lib.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if v := versionDirs(t, dir, pdf.SourceID); len(v) != 1 {
		t.Fatalf("versions = %v, want one copy", v)
	}
	copyDir, err := lib.Dir(ctx, pdf.SourceID)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(copyDir, "SKILL.md")); err != nil || !strings.Contains(string(b), "Fill PDF forms") {
		t.Fatalf("copied SKILL.md: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(copyDir, "scripts", "fill.sh"))
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("script mode = %v, %v", info.Mode(), err)
		}
	}
	if s := discovered(t, lib, "pdf"); s.ImportStatus != importStatusImported {
		t.Fatalf("status after import = %q", s.ImportStatus)
	}
	if err := lib.Remove(ctx, pdf.SourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, storeDir, pdf.SourceID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("copy should be gone: %v", err)
	}
	if err := lib.Remove(ctx, pdf.SourceID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("second remove = %v", err)
	}
	if after := snap(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("source changed:\nbefore %v\nafter  %v", before, after)
	}
	if want := []auditCall{{"skill.import", "skill:" + pdf.SourceID}, {"skill.remove", "skill:" + pdf.SourceID}}; !reflect.DeepEqual(*calls, want) {
		t.Fatalf("audit = %v", *calls)
	}
}

func TestImportUpdateNeedsReplaceAndSwapsVersion(t *testing.T) {
	root := t.TempDir()
	md := filepath.Join(root, "pdf", "SKILL.md")
	writeFile(t, md, skillMD("pdf", "v1", "MIT"))
	lib, dir, _ := newLibrary(t, Root{Origin: OriginClaude, Path: root})
	ctx := context.Background()
	v1 := discovered(t, lib, "pdf")
	first, _, err := lib.Import(ctx, ImportRequest{SourceID: v1.SourceID, Hash: v1.Hash})
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, md, skillMD("pdf", "v2", "MIT"))
	if _, _, err := lib.Import(ctx, ImportRequest{SourceID: v1.SourceID, Hash: v1.Hash}); !errors.Is(err, ErrChanged) {
		t.Fatalf("stale hash = %v, want ErrChanged", err)
	}
	v2 := discovered(t, lib, "pdf")
	if v2.ImportStatus != importStatusUpdateReady || v2.SourceID != v1.SourceID {
		t.Fatalf("v2 = %+v", v2)
	}
	if _, _, err := lib.Import(ctx, ImportRequest{SourceID: v2.SourceID, Hash: v2.Hash}); !errors.Is(err, ErrReplaceRequired) {
		t.Fatalf("update without replace = %v", err)
	}
	got, result, err := lib.Import(ctx, ImportRequest{SourceID: v2.SourceID, Hash: v2.Hash, Replace: true})
	if err != nil || result != ResultReplaced {
		t.Fatalf("replace: %v %q", err, result)
	}
	if got.Description != "v2" || got.ImportedAt != first.ImportedAt || got.Hash == first.Hash {
		t.Fatalf("replaced = %+v", got)
	}
	versions := versionDirs(t, dir, v1.SourceID)
	if len(versions) != 1 || versions[0] != strings.TrimPrefix(v2.Hash, "sha256:")[:versionChars] {
		t.Fatalf("versions = %v", versions)
	}
	list, _ := lib.List(ctx)
	if len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
}

func TestImportWithoutLicenseNeedsConfirmation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "bare", "SKILL.md"), skillMD("bare", "x", ""))
	lib, _, _ := newLibrary(t, Root{Origin: OriginCodex, Path: root})
	ctx := context.Background()
	bare := discovered(t, lib, "bare")
	if _, _, err := lib.Import(ctx, ImportRequest{SourceID: bare.SourceID, Hash: bare.Hash}); !errors.Is(err, ErrLicenseConfirmation) {
		t.Fatalf("err = %v", err)
	}
	if list, _ := lib.List(ctx); len(list) != 0 {
		t.Fatal("a refused import must not be recorded")
	}
	got, _, err := lib.Import(ctx, ImportRequest{SourceID: bare.SourceID, Hash: bare.Hash, ConfirmLicense: true})
	if err != nil || !got.LicenseConfirmed {
		t.Fatalf("confirmed import = %+v, %v", got, err)
	}
}

func TestImportRejectsUnknownSource(t *testing.T) {
	lib, _, _ := newLibrary(t, Root{Origin: OriginClaude, Path: t.TempDir()})
	_, _, err := lib.Import(context.Background(), ImportRequest{SourceID: strings.Repeat("a", 32), Hash: "sha256:x"})
	if !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := lib.Import(context.Background(), ImportRequest{}); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("empty request = %v", err)
	}
}

func TestNewSweepsLeftoversAndRejectsBadRoots(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pdf", "SKILL.md"), skillMD("pdf", "x", "MIT"))
	lib, dir, _ := newLibrary(t, Root{Origin: OriginClaude, Path: root})
	pdf := discovered(t, lib, "pdf")
	if _, _, err := lib.Import(context.Background(), ImportRequest{SourceID: pdf.SourceID, Hash: pdf.Hash}); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(dir, storeDir, strings.Repeat("b", 32))
	writeFile(t, filepath.Join(orphan, "x", "SKILL.md"), "orphan")
	stale := filepath.Join(dir, storeDir, pdf.SourceID, ".tmp-123")
	writeFile(t, filepath.Join(stale, "SKILL.md"), "partial")
	writeFile(t, filepath.Join(dir, indexFile+".abc.tmp"), "{}")

	if _, err := New(Config{Dir: dir, Roots: []Root{{Origin: OriginClaude, Path: root}}, Now: time.Now}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{orphan, stale, filepath.Join(dir, indexFile+".abc.tmp")} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s should be swept: %v", p, err)
		}
	}
	if v := versionDirs(t, dir, pdf.SourceID); len(v) != 1 {
		t.Fatalf("active copy must survive the sweep: %v", v)
	}

	bad := []Config{
		{Dir: "relative", Now: time.Now},
		{Dir: dir, Now: time.Now, Roots: []Root{{Origin: "other", Path: root}}},
		{Dir: dir, Now: time.Now, Roots: []Root{{Origin: OriginClaude, Path: "rel"}}},
		{Dir: dir, Now: time.Now, Roots: []Root{{Origin: OriginClaude, Path: filepath.Dir(dir)}}},
	}
	for i, cfg := range bad {
		if _, err := New(cfg); err == nil {
			t.Errorf("config %d should be rejected", i)
		}
	}
}

func TestCorruptIndexIsAnError(t *testing.T) {
	lib, dir, _ := newLibrary(t)
	writeFile(t, filepath.Join(dir, indexFile), `{"version":1,"skills":[{"id":"../../etc","version":"x"}]}`)
	if _, err := lib.List(context.Background()); err == nil {
		t.Fatal("a malformed index entry must be refused")
	}
}
