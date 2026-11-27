//go:build unix

package pty

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// unixProcess is a program started by creack/pty. The child is a session leader
// (setsid) with the terminal as its controlling tty, so its process group and session
// both carry its pid.
type unixProcess struct {
	cmd    *exec.Cmd
	master *os.File
	done   chan struct{}
	code   int

	closeOnce sync.Once
	closeErr  error
}

func start(opts Options) (Process, error) {
	cmd := exec.Command(opts.Path, opts.Args...)
	cmd.Dir = opts.Dir
	// A non-nil, possibly empty, Env stops exec from inheriting the daemon environment.
	cmd.Env = append([]string{}, opts.Env...)
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: opts.Cols, Rows: opts.Rows})
	if err != nil {
		return nil, fmt.Errorf("pty: start %s: %w", opts.Path, err)
	}
	p := &unixProcess{cmd: cmd, master: master, done: make(chan struct{}), code: -1}
	go p.wait()
	return p, nil
}

func (p *unixProcess) wait() {
	err := p.cmd.Wait()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		p.code = 0
	case errors.As(err, &exitErr):
		p.code = exitErr.ExitCode()
	}
	close(p.done)
}

func (p *unixProcess) Read(b []byte) (int, error)  { return p.master.Read(b) }
func (p *unixProcess) Write(b []byte) (int, error) { return p.master.Write(b) }
func (p *unixProcess) Pid() int                    { return p.cmd.Process.Pid }
func (p *unixProcess) Done() <-chan struct{}       { return p.done }

func (p *unixProcess) ExitCode() int {
	<-p.done
	return p.code
}

func (p *unixProcess) Resize(cols, rows uint16) error {
	if err := validateSize(cols, rows); err != nil {
		return err
	}
	return pty.Setsize(p.master, &pty.Winsize{Cols: cols, Rows: rows})
}

// Close kills the tree first, then closes the master. Closing the master alone would
// only send SIGHUP, which a program may ignore.
func (p *unixProcess) Close() error {
	p.closeOnce.Do(func() {
		killTree(p.cmd.Process.Pid)
		p.closeErr = p.master.Close()
	})
	return p.closeErr
}

// killTree sends SIGKILL to the leader's process group, to every process still in its
// session and to every descendant found by parent id. Interactive shells move each job
// to its own process group, so the group alone is not enough. Two passes catch children
// forked while the first pass ran. A process that called setsid itself and was
// reparented leaves both nets; that is a known limit without cgroups.
func killTree(leader int) {
	for range 2 {
		_ = syscall.Kill(-leader, syscall.SIGKILL)
		for _, pid := range relatedProcesses(leader) {
			if pid > 1 && pid != os.Getpid() {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	}
}

// descendants returns every pid below root in the parent map.
func descendants(root int, parent map[int]int) []int {
	children := map[int][]int{}
	for pid, ppid := range parent {
		children[ppid] = append(children[ppid], pid)
	}
	var out []int
	queue := []int{root}
	seen := map[int]bool{root: true}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		for _, c := range children[next] {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
				queue = append(queue, c)
			}
		}
	}
	return out
}
