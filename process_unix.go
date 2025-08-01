//go:build linux || darwin

// Package processctrl Unix implementation
//
// This file contains Unix-specific process control functionality using
// POSIX signals for process suspension, resumption, and termination.

package processctrl

import (
	"fmt"
	"syscall"
	"time"
)

// pauseUnix implements Unix-specific process suspension using SIGSTOP.
func (p *Process) pauseUnix() error {
	if err := p.cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		return fmt.Errorf("failed to pause process: %w", err)
	}
	return nil
}

// pauseImpl provides the cross-platform interface for Unix.
func (p *Process) pauseImpl() error {
	return p.pauseUnix()
}

// resumeUnix implements Unix-specific process resumption using SIGCONT.
func (p *Process) resumeUnix() error {
	if err := p.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("failed to resume process: %w", err)
	}
	return nil
}

// resumeImpl provides the cross-platform interface for Unix.
func (p *Process) resumeImpl() error {
	return p.resumeUnix()
}

// killWithSignalUnix implements graceful and forceful process termination for Unix systems.
// When graceful is true, it first sends SIGTERM to allow the process to clean up,
// then waits for the specified timeout before sending SIGKILL if needed.
//
// Parameters:
//   - timeout: Maximum time to wait for graceful shutdown before force-killing
//   - graceful: If true, attempts SIGTERM before SIGKILL; if false, uses SIGKILL immediately
//
// Returns an error if the termination operation fails.
func (p *Process) killWithSignalUnix(timeout time.Duration, graceful bool) error {
	if graceful {
		// Try SIGTERM first
		if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- p.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
			return nil
		case <-time.After(timeout):
			// Timeout, force kill
		}
	}

	// Force kill with SIGKILL
	return p.cmd.Process.Kill()
}

// killWithSignalImpl provides the cross-platform interface for Unix.
func (p *Process) killWithSignalImpl(timeout time.Duration, graceful bool) error {
	return p.killWithSignalUnix(timeout, graceful)
}
