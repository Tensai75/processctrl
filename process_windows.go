//go:build windows

// Package processctrl Windows implementation
//
// This file contains Windows-specific process control functionality using
// Windows NT API calls for process suspension, resumption, and termination.
//
// Some code copied from https://github.com/shirou/gopsutil/blob/master/process/process_windows.go
// and modified for use in processctrl. The gopsutil license is provided below:
//
// gopsutil is distributed under BSD license reproduced below.
//
// Copyright (c) 2014, WAKAYAMA Shirou
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
//   - Redistributions of source code must retain the above copyright notice, this
//     list of conditions and the following disclaimer.
//   - Redistributions in binary form must reproduce the above copyright notice,
//     this list of conditions and the following disclaimer in the documentation
//     and/or other materials provided with the distribution.
//   - Neither the name of the gopsutil authors nor the names of its contributors
//     may be used to endorse or promote products derived from this software without
//     specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package processctrl

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

// Windows NT API function declarations for process suspension/resumption.
// These functions are loaded dynamically from ntdll.dll.
var (
	modntdll             = windows.NewLazySystemDLL("ntdll.dll")
	procNtResumeProcess  = modntdll.NewProc("NtResumeProcess")
	procNtSuspendProcess = modntdll.NewProc("NtSuspendProcess")
)

// Pause suspends the process execution using Windows NT API NtSuspendProcess.
// The process can be resumed later using Resume().
// This method is thread-safe and uses Windows-specific process suspension.
//
// Returns an error if:
//   - The process is not running
//   - The process is already paused
//   - Failed to open the process handle
//   - NtSuspendProcess API call fails
func (p *Process) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.paused {
		return fmt.Errorf("process not running or already paused")
	}

	h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, uint32(p.cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("failed to open process for suspend: %w", err)
	}
	defer func() { _ = windows.CloseHandle(h) }() // Ignore error on cleanup

	r1, _, _ := procNtSuspendProcess.Call(uintptr(h))
	if r1 != 0 {
		// See https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/596a1078-e883-4972-9bbc-49e60bebca55
		return fmt.Errorf("NtSuspendProcess failed: status=0x%.8X", r1)
	}

	p.paused = true
	return nil
}

// Resume continues the execution of a paused process using Windows NT API NtResumeProcess.
// This method is thread-safe and uses Windows-specific process resumption.
//
// Returns an error if:
//   - The process is not running
//   - The process is not currently paused
//   - Failed to open the process handle
//   - NtResumeProcess API call fails
func (p *Process) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || !p.paused {
		return fmt.Errorf("process not running or not paused")
	}

	h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, uint32(p.cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("failed to open process for resume: %w", err)
	}
	defer func() { _ = windows.CloseHandle(h) }() // Ignore error on cleanup

	r1, _, _ := procNtResumeProcess.Call(uintptr(h))
	if r1 != 0 {
		// See https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/596a1078-e883-4972-9bbc-49e60bebca55
		return fmt.Errorf("NtResumeProcess failed: status=0x%.8X", r1)
	}

	p.paused = false
	return nil
}

// killWithSignal implements graceful and forceful process termination for Windows systems.
// Unlike Unix systems that use signals, Windows uses TerminateProcess API calls.
// When graceful is true, it first tries TerminateProcess with exit code 1,
// then waits for the specified timeout before using Kill() if needed.
//
// Parameters:
//   - timeout: Maximum time to wait for graceful shutdown before force-killing
//   - graceful: If true, attempts gentle termination before force-kill; if false, uses Kill() immediately
//
// Returns an error if the termination operation fails.
func (p *Process) killWithSignal(timeout time.Duration, graceful bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("process is not running")
	}

	if graceful {
		// On Windows, we don't have SIGTERM, so we'll try a gentle approach
		// by giving the process a chance to exit gracefully
		done := make(chan error, 1)
		go func() {
			done <- p.cmd.Wait()
		}()

		// First, try to close the process gracefully by terminating it
		// This is less forceful than Kill()
		h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(p.cmd.Process.Pid))
		if err == nil {
			_ = windows.TerminateProcess(h, 1) // Ignore error - process might already be dead
			_ = windows.CloseHandle(h)         // Ignore error on cleanup
		}

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

	// Force kill
	err := p.cmd.Process.Kill()
	if err == nil {
		p.running = false
		p.paused = false
	}
	return err
}
