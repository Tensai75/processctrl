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

// Pause suspends the process execution using SIGSTOP signal.
// The process can be resumed later using Resume().
// This method is thread-safe and uses POSIX signals for process control.
//
// Returns an error if:
//   - The process is not running
//   - The process is already paused
//   - The SIGSTOP signal fails to be sent
func (p *Process) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running || p.paused {
		return fmt.Errorf("process not running or already paused")
	}
	if err := p.cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		return fmt.Errorf("failed to pause process: %w", err)
	}
	p.paused = true
	return nil
}

// Resume continues the execution of a paused process using SIGCONT signal.
// This method is thread-safe and uses POSIX signals for process control.
//
// Returns an error if:
//   - The process is not running
//   - The process is not currently paused
//   - The SIGCONT signal fails to be sent
func (p *Process) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running || !p.paused {
		return fmt.Errorf("process not running or not paused")
	}
	if err := p.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("failed to resume process: %w", err)
	}
	p.paused = false
	return nil
}

// killWithSignal implements graceful and forceful process termination for Unix systems.
// When graceful is true, it first sends SIGTERM to allow the process to clean up,
// then waits for the specified timeout before sending SIGKILL if needed.
//
// Parameters:
//   - timeout: Maximum time to wait for graceful shutdown before force-killing
//   - graceful: If true, attempts SIGTERM before SIGKILL; if false, uses SIGKILL immediately
//
// Returns an error if the termination operation fails.
func (p *Process) killWithSignal(timeout time.Duration, graceful bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("process is not running")
	}

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
			p.running = false
			p.paused = false
			return nil
		case <-time.After(timeout):
			// Timeout, force kill
		}
	}

	// Force kill with SIGKILL
	err := p.cmd.Process.Kill()
	if err == nil {
		p.running = false
		p.paused = false
	}
	return err
}
