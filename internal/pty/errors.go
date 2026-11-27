package pty

import "errors"

// Sentinel errors. The API maps each one to a stable error code.
var (
	// ErrInvalidInput covers a bad size, a working directory outside the allowed
	// root, or a directory that does not exist.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound means no session with that id exists for the caller.
	ErrNotFound = errors.New("terminal session not found")
	// ErrBusy means another client is already attached to the session.
	ErrBusy = errors.New("terminal session already has a client attached")
	// ErrLimit means the node already runs the maximum number of sessions.
	ErrLimit = errors.New("terminal session limit reached")
	// ErrDenied means the start policy refused the session.
	ErrDenied = errors.New("terminal session refused by policy")
	// ErrClosed means the manager is shutting down.
	ErrClosed = errors.New("terminal manager closed")
)
