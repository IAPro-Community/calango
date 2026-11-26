//go:build darwin

package pty

import "golang.org/x/sys/unix"

// relatedProcesses lists leader's descendants by parent id and, where getsid allows,
// the processes still in its session.
func relatedProcesses(leader int) []int {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil
	}
	parent := make(map[int]int, len(procs))
	var out []int
	for i := range procs {
		pid := int(procs[i].Proc.P_pid)
		if pid == leader {
			continue
		}
		parent[pid] = int(procs[i].Eproc.Ppid)
		if sid, err := unix.Getsid(pid); err == nil && sid == leader {
			out = append(out, pid)
		}
	}
	return append(out, descendants(leader, parent)...)
}
