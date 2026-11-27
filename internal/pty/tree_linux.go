//go:build linux

package pty

import (
	"bytes"
	"os"
	"strconv"
)

// relatedProcesses lists the processes in leader's session plus its descendants,
// read from /proc. The session survives the leader's exit, so this also finds
// background jobs a shell left behind.
func relatedProcesses(leader int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	parent := map[int]int{}
	var out []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == leader {
			continue
		}
		ppid, sid, ok := readStat(pid)
		if !ok {
			continue
		}
		parent[pid] = ppid
		if sid == leader {
			out = append(out, pid)
		}
	}
	return append(out, descendants(leader, parent)...)
}

// readStat returns the parent and session ids from /proc/<pid>/stat. The command name
// may hold spaces and parentheses, so fields are counted after the last ')'.
func readStat(pid int) (ppid, sid int, ok bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, 0, false
	}
	i := bytes.LastIndexByte(data, ')')
	if i < 0 {
		return 0, 0, false
	}
	// state ppid pgrp session ...
	fields := bytes.Fields(data[i+1:])
	if len(fields) < 4 {
		return 0, 0, false
	}
	ppid, err1 := strconv.Atoi(string(fields[1]))
	sid, err2 := strconv.Atoi(string(fields[3]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return ppid, sid, true
}
