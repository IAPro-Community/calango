//go:build unix && !linux && !darwin

package pty

// relatedProcesses has no process table reader on this platform yet; killTree then
// relies on the process group alone.
func relatedProcesses(int) []int { return nil }
