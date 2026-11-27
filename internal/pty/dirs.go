package pty

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DirResolver turns the directory a client asked for into the absolute directory the
// shell starts in, or refuses it with ErrInvalidInput. It is the hook for per-task
// worktrees (F4-T02, issue #83): a resolver that maps a task to its worktree can
// replace RootDir without touching Manager.
type DirResolver interface {
	ResolveDir(ctx context.Context, ownerID, requested string) (string, error)
}

// RootDir allows the root directory and anything below it, after symlinks are
// resolved. An empty request means the root itself. Relative requests are taken
// from the root; absolute ones must already lie inside it.
type RootDir struct {
	root string
}

// NewRootDir resolves root once. It must exist and be a directory.
func NewRootDir(root string) (*RootDir, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("pty: root directory: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("pty: root directory: %w", err)
	}
	st, err := os.Stat(real)
	if err != nil {
		return nil, fmt.Errorf("pty: root directory: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("pty: root %s is not a directory", real)
	}
	return &RootDir{root: real}, nil
}

// Root returns the resolved root directory.
func (d *RootDir) Root() string { return d.root }

// ResolveDir implements DirResolver.
func (d *RootDir) ResolveDir(_ context.Context, _ string, requested string) (string, error) {
	if strings.ContainsRune(requested, 0) {
		return "", fmt.Errorf("%w: directory holds NUL", ErrInvalidInput)
	}
	target := d.root
	if requested != "" {
		if filepath.IsAbs(requested) {
			target = filepath.Clean(requested)
		} else {
			target = filepath.Join(d.root, requested)
		}
	}
	real, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("%w: directory does not exist or cannot be read", ErrInvalidInput)
	}
	if !within(d.root, real) {
		return "", fmt.Errorf("%w: directory is outside the terminal workspace", ErrInvalidInput)
	}
	st, err := os.Stat(real)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("%w: not a directory", ErrInvalidInput)
	}
	return real, nil
}

// within reports whether path is root or below it. Both must be clean absolute paths.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}
