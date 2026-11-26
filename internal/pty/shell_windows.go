//go:build windows

package pty

import (
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultShell returns Windows PowerShell when it is on PATH, else %ComSpec%, else
// cmd.exe under %SystemRoot%.
func DefaultShell() (string, []string) {
	if ps, err := exec.LookPath("powershell.exe"); err == nil {
		if abs, err := filepath.Abs(ps); err == nil {
			return abs, []string{"-NoLogo"}
		}
	}
	if comspec := os.Getenv("ComSpec"); filepath.IsAbs(comspec) {
		return comspec, nil
	}
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"), nil
}
