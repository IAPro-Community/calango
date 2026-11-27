//go:build unix

package pty

import (
	"os"
	"path/filepath"
)

// DefaultShell returns the person's login shell from $SHELL when it is an absolute path
// to an executable file, else /bin/sh. It runs as a login shell so the profile sets PATH
// the way a normal terminal would.
func DefaultShell() (string, []string) {
	if sh := os.Getenv("SHELL"); filepath.IsAbs(sh) {
		if st, err := os.Stat(sh); err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0 {
			return sh, []string{"-l"}
		}
	}
	return "/bin/sh", []string{"-l"}
}
