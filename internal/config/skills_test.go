package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSkillRootsDefaultToClaudeAndCodexHomes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CALANGO_SKILL_DIRS", "")
	if err := os.Unsetenv("CALANGO_SKILL_DIRS"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", "")

	got, err := loadSkillRoots()
	if err != nil {
		t.Fatal(err)
	}
	want := []SkillRoot{
		{Origin: "claude", Path: filepath.Join(home, ".claude", "skills")},
		{Origin: "codex", Path: filepath.Join(home, ".codex", "skills")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %+v, want %+v", got, want)
	}

	codexHome := filepath.Join(t.TempDir(), "codex")
	t.Setenv("CODEX_HOME", codexHome)
	got, err = loadSkillRoots()
	if err != nil {
		t.Fatal(err)
	}
	if got[1].Path != filepath.Join(codexHome, "skills") {
		t.Fatalf("codex root = %q, want CODEX_HOME/skills", got[1].Path)
	}
}

func TestSkillRootsExplicitListReplacesDefaults(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	sep := string(os.PathListSeparator)
	t.Setenv("CALANGO_SKILL_DIRS", "project="+a+sep+" claude="+b+sep+"project="+a+sep)
	got, err := loadSkillRoots()
	if err != nil {
		t.Fatal(err)
	}
	want := []SkillRoot{{Origin: "project", Path: a}, {Origin: "claude", Path: b}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %+v, want %+v", got, want)
	}

	t.Setenv("CALANGO_SKILL_DIRS", "")
	if got, err := loadSkillRoots(); err != nil || len(got) != 0 {
		t.Fatalf("empty value should disable discovery: %+v, %v", got, err)
	}
}

func TestSkillRootsRejectMalformedEntries(t *testing.T) {
	abs := t.TempDir()
	sep := string(os.PathListSeparator)
	many := make([]string, 0, maxSkillRoots+1)
	for i := 0; i <= maxSkillRoots; i++ {
		many = append(many, "project="+filepath.Join(abs, strings.Repeat("d", i+1)))
	}
	for _, raw := range []string{
		abs,
		"other=" + abs,
		"claude=relative/path",
		strings.Join(many, sep),
	} {
		if _, err := parseSkillRoots(raw); err == nil {
			t.Errorf("parseSkillRoots(%q) should fail", raw)
		}
	}
}
