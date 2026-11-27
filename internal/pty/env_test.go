package pty

import (
	"strings"
	"testing"
)

func TestFilterEnvDropsSecretsAndKeepsBasics(t *testing.T) {
	got := FilterEnv([]string{
		"PATH=/usr/bin", "HOME=/home/a", "LC_ALL=C.UTF-8",
		"CALANGO_VAULT_KEY=s3cret", "CALANGO_VAULT_KEY_FILE=/k", "calango_addr=x",
		"ANTHROPIC_API_KEY=sk-1", "GITHUB_TOKEN=gh-1", "OPENAI_API_KEY=sk-2",
		"AWS_SECRET_ACCESS_KEY=a", "TERM=dumb", "NOEQUALS", "=bad",
	})
	joined := strings.Join(got, "\n")
	for _, leaked := range []string{"s3cret", "CALANGO", "calango", "sk-1", "gh-1", "sk-2", "AWS_", "TERM=dumb", "NOEQUALS"} {
		if strings.Contains(joined, leaked) {
			t.Fatalf("filtered env leaks %q:\n%s", leaked, joined)
		}
	}
	for _, kept := range []string{"PATH=/usr/bin", "HOME=/home/a", "LC_ALL=C.UTF-8"} {
		if !strings.Contains(joined, kept) {
			t.Fatalf("filtered env lost %q:\n%s", kept, joined)
		}
	}
}

func TestValidIDMatchesGeneratedIDs(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id, err := newSessionID()
		if err != nil {
			t.Fatal(err)
		}
		if !ValidID(id) {
			t.Fatalf("generated id %q is not valid", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
	for _, bad := range []string{"", "abc", strings.Repeat("a", 42), strings.Repeat("*", 43), "../" + strings.Repeat("a", 40)} {
		if ValidID(bad) {
			t.Fatalf("ValidID(%q) = true", bad)
		}
	}
}

func TestAppendBoundedKeepsTheTail(t *testing.T) {
	buf := appendBounded(nil, []byte("hello"), 8)
	buf = appendBounded(buf, []byte("world"), 8)
	if string(buf) != "lloworld" {
		t.Fatalf("buf = %q", buf)
	}
}
