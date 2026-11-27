// Package pty runs interactive shells on a pseudo-terminal for the integrated terminal
// (F4-T03). Start is the platform layer: creack/pty on Unix and ConPTY on Windows.
// Manager keeps the sessions in memory, bounds how many may run, expires idle ones and
// kills the whole process tree when a session ends or the daemon stops.
package pty

import (
	"errors"
	"fmt"
	"io"
)

// Terminal size bounds. A client that asks for more is refused, not clamped.
const (
	DefaultCols uint16 = 80
	DefaultRows uint16 = 24
	MaxCols     uint16 = 1000
	MaxRows     uint16 = 1000
)

// Options describe one process to start on a new pseudo-terminal.
type Options struct {
	// Path is the absolute path of the program. Args do not repeat it.
	Path string
	Args []string
	// Dir is the working directory. Manager resolves and checks it before Start.
	Dir string
	// Env is the full environment of the child, as KEY=value entries. Nothing is
	// inherited from the daemon beyond what is listed here.
	Env        []string
	Cols, Rows uint16
}

// Process is a program running on a pseudo-terminal. Read returns what the program
// writes to the terminal and Write types into it.
type Process interface {
	io.ReadWriter
	// Resize changes the terminal size seen by the program.
	Resize(cols, rows uint16) error
	// Pid is the operating system id of the program that was started.
	Pid() int
	// Done is closed when the program that was started exits. Children it left behind
	// may still run until Close.
	Done() <-chan struct{}
	// ExitCode is the program's exit status once Done is closed; -1 when it was
	// killed or the status is unknown.
	ExitCode() int
	// Close kills the program and every process it started, then releases the
	// terminal. A Read in progress returns an error. Close is idempotent.
	Close() error
}

// Start runs opts on a new pseudo-terminal.
func Start(opts Options) (Process, error) {
	if err := validateSize(opts.Cols, opts.Rows); err != nil {
		return nil, err
	}
	if opts.Path == "" {
		return nil, errors.New("pty: program path is required")
	}
	return start(opts)
}

func validateSize(cols, rows uint16) error {
	if cols == 0 || rows == 0 || cols > MaxCols || rows > MaxRows {
		return fmt.Errorf("%w: terminal size must be 1..%d columns and 1..%d rows", ErrInvalidInput, MaxCols, MaxRows)
	}
	return nil
}
