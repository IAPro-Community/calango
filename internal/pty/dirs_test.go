package pty

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRootDirAllowsOnlyTheWorkspace(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "workspace")
	sub := filepath.Join(root, "proj", "src")
	outside := filepath.Join(base, "outside")
	for _, d := range []string{sub, outside} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	symlinkOK := os.Symlink(outside, link) == nil

	dirs, err := NewRootDir(root)
	if err != nil {
		t.Fatal(err)
	}
	realSub, _ := filepath.EvalSymlinks(sub)
	ctx := context.Background()
	for _, req := range []string{"", ".", "proj/src", realSub, filepath.Join("proj", "..", "proj", "src")} {
		got, err := dirs.ResolveDir(ctx, "owner", req)
		if err != nil {
			t.Fatalf("ResolveDir(%q): %v", req, err)
		}
		if !within(dirs.Root(), got) {
			t.Fatalf("ResolveDir(%q) = %q, outside root", req, got)
		}
	}
	bad := []string{"..", "../outside", outside, base, "missing", "file.txt", filepath.VolumeName(base) + string(filepath.Separator)}
	if symlinkOK {
		bad = append(bad, "escape")
	}
	for _, req := range bad {
		if _, err := dirs.ResolveDir(ctx, "owner", req); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("ResolveDir(%q) err = %v, want ErrInvalidInput", req, err)
		}
	}
}
