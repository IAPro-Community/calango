//go:build windows

package pty

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// conPTY is a program attached to a Windows pseudo console (ConPTY, Windows 10 1809+).
// The program runs inside a job object with KILL_ON_JOB_CLOSE, so closing the job ends
// every process it started. It is built on golang.org/x/sys/windows, which the module
// already depends on, instead of a separate ConPTY library.
type conPTY struct {
	console windows.Handle
	input   *os.File // our write end of the console input pipe
	output  *os.File // our read end of the console output pipe
	process windows.Handle
	job     windows.Handle
	pid     int
	done    chan struct{}
	code    int

	consoleOnce sync.Once
	closeOnce   sync.Once
}

func start(opts Options) (_ Process, err error) {
	var inRead, inWrite, outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("pty: input pipe: %w", err)
	}
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		closeHandles(inRead, inWrite)
		return nil, fmt.Errorf("pty: output pipe: %w", err)
	}
	var console windows.Handle
	size := windows.Coord{X: int16(opts.Cols), Y: int16(opts.Rows)}
	err = windows.CreatePseudoConsole(size, inRead, outWrite, 0, &console)
	// The console duplicated its ends of the pipes; ours are no longer needed.
	closeHandles(inRead, outWrite)
	if err != nil {
		closeHandles(inWrite, outRead)
		return nil, fmt.Errorf("pty: create pseudo console: %w", err)
	}
	p := &conPTY{
		console: console,
		input:   os.NewFile(uintptr(inWrite), "conpty-input"),
		output:  os.NewFile(uintptr(outRead), "conpty-output"),
		done:    make(chan struct{}),
		code:    -1,
	}
	defer func() {
		if err != nil {
			p.closeConsole()
			_ = p.input.Close()
			_ = p.output.Close()
			if p.job != 0 {
				_ = windows.CloseHandle(p.job)
			}
		}
	}()

	job, err := newKillOnCloseJob()
	if err != nil {
		return nil, err
	}
	p.job = job

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("pty: attribute list: %w", err)
	}
	defer attrs.Delete()
	// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE takes the HPCON value itself, not a pointer to it.
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&console)), unsafe.Sizeof(console)); err != nil {
		return nil, fmt.Errorf("pty: attach pseudo console: %w", err)
	}

	si := windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(si))
	// Without explicit invalid std handles a child of a daemon with redirected stdio
	// would write to the daemon's pipes instead of the pseudo console.
	si.Flags = windows.STARTF_USESTDHANDLES
	si.StdInput = windows.InvalidHandle
	si.StdOutput = windows.InvalidHandle
	si.StdErr = windows.InvalidHandle

	cmdLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{opts.Path}, opts.Args...)))
	if err != nil {
		return nil, fmt.Errorf("%w: command line: %v", ErrInvalidInput, err)
	}
	dir, err := windows.UTF16PtrFromString(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("%w: directory: %v", ErrInvalidInput, err)
	}
	env, err := envBlock(opts.Env)
	if err != nil {
		return nil, err
	}
	// Suspended until it is in the job, so it cannot start a child outside the job.
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_SUSPENDED)
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, cmdLine, nil, nil, false, flags, &env[0], dir, &si.StartupInfo, &pi); err != nil {
		return nil, fmt.Errorf("pty: start %s: %w", opts.Path, err)
	}
	if err := windows.AssignProcessToJobObject(job, pi.Process); err != nil {
		_ = windows.TerminateProcess(pi.Process, 1)
		closeHandles(pi.Thread, pi.Process)
		return nil, fmt.Errorf("pty: assign job: %w", err)
	}
	if _, err := windows.ResumeThread(pi.Thread); err != nil {
		_ = windows.TerminateJobObject(job, 1)
		closeHandles(pi.Thread, pi.Process)
		return nil, fmt.Errorf("pty: resume: %w", err)
	}
	_ = windows.CloseHandle(pi.Thread)
	p.process = pi.Process
	p.pid = int(pi.ProcessId)
	go p.wait()
	return p, nil
}

// newKillOnCloseJob returns a job object that kills its processes when its last
// handle closes, including when the daemon itself dies.
func newKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("pty: job object: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, fmt.Errorf("pty: job limits: %w", err)
	}
	return job, nil
}

// wait records the exit status. The console keeps its output pipe open after the
// program exits, so it is closed here to let Read reach EOF once output is drained.
func (p *conPTY) wait() {
	if _, err := windows.WaitForSingleObject(p.process, windows.INFINITE); err == nil {
		var code uint32
		if windows.GetExitCodeProcess(p.process, &code) == nil {
			p.code = int(code)
		}
	}
	close(p.done)
	go p.closeConsole()
}

func (p *conPTY) closeConsole() {
	p.consoleOnce.Do(func() { windows.ClosePseudoConsole(p.console) })
}

func (p *conPTY) Read(b []byte) (int, error) {
	n, err := p.output.Read(b)
	if errors.Is(err, windows.ERROR_BROKEN_PIPE) {
		return n, os.ErrClosed
	}
	return n, err
}

func (p *conPTY) Write(b []byte) (int, error) { return p.input.Write(b) }
func (p *conPTY) Pid() int                    { return p.pid }
func (p *conPTY) Done() <-chan struct{}       { return p.done }

func (p *conPTY) ExitCode() int {
	<-p.done
	return p.code
}

func (p *conPTY) Resize(cols, rows uint16) error {
	if err := validateSize(cols, rows); err != nil {
		return err
	}
	return windows.ResizePseudoConsole(p.console, windows.Coord{X: int16(cols), Y: int16(rows)})
}

// Close ends every process in the job, then the console, then our pipe ends.
func (p *conPTY) Close() error {
	p.closeOnce.Do(func() {
		_ = windows.TerminateJobObject(p.job, 1)
		_ = windows.CloseHandle(p.job)
		_ = p.input.Close()
		p.closeConsole()
		_ = p.output.Close()
		go func() {
			<-p.done
			_ = windows.CloseHandle(p.process)
		}()
	})
	return nil
}

// envBlock encodes env as a CREATE_UNICODE_ENVIRONMENT block: KEY=value entries, each
// NUL-terminated, followed by one more NUL.
func envBlock(env []string) ([]uint16, error) {
	var block []uint16
	for _, kv := range env {
		for _, r := range kv {
			if r == 0 {
				return nil, fmt.Errorf("%w: environment entry holds NUL", ErrInvalidInput)
			}
		}
		block = append(block, utf16.Encode([]rune(kv))...)
		block = append(block, 0)
	}
	if len(block) == 0 {
		block = append(block, 0)
	}
	return append(block, 0), nil
}

func closeHandles(handles ...windows.Handle) {
	for _, h := range handles {
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
	}
}
