package processctrl

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	windowsOS           = "windows"
	testBufferSize      = 5
	testTimeout         = 5
	longRunningPingTime = 10
)

func TestNew(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo hello")
	} else {
		proc = New("echo", "hello")
	}

	if proc == nil {
		t.Fatal("New() returned nil")
	}
	if proc.program == "" {
		t.Fatal("program not set")
	}
}

func TestNewWithBuffer(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = NewWithBuffer(testBufferSize, "cmd", "/c", "echo hello")
	} else {
		proc = NewWithBuffer(testBufferSize, "echo", "hello")
	}

	if proc == nil {
		t.Fatal("NewWithBuffer() returned nil")
	}
	if proc.program == "" {
		t.Fatal("program not set")
	}
}

func TestBasicRun(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo hello world")
	} else {
		proc = New("echo", "hello world")
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	var output []string
	go func() {
		for line := range stderr {
			t.Logf("STDERR: %s", line)
		}
	}()

	for line := range stdout {
		output = append(output, line)
	}

	if len(output) == 0 {
		t.Fatal("No output received")
	}

	if !strings.Contains(strings.Join(output, " "), "hello world") {
		t.Errorf("Expected 'hello world' in output, got: %v", output)
	}

	// Process may have already finished by the time we get here
	if proc.IsRunning() {
		state, err := proc.Wait()
		if err != nil {
			t.Fatalf("Wait() failed: %v", err)
		}
		if state != nil && state.ExitCode() != 0 {
			t.Errorf("Expected exit code 0, got %d", state.ExitCode())
		}
	}
}

func TestRunWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout*time.Second)
	defer cancel()

	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo context test")
	} else {
		proc = New("echo", "context test")
	}

	stdout, stderr, err := proc.RunWithContext(ctx)
	if err != nil {
		t.Fatalf("RunWithContext() failed: %v", err)
	}

	go func() {
		for range stderr {
			// consume stderr
		}
	}()

	var output []string
	for line := range stdout {
		output = append(output, line)
	}

	if len(output) == 0 {
		t.Fatal("No output received from context run")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Use a command that runs longer
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("ping", "-n", "10", "localhost")
	} else {
		proc = New("sleep", "10")
	}

	stdout, stderr, err := proc.RunWithContext(ctx)
	if err != nil {
		t.Fatalf("RunWithContext() failed: %v", err)
	}

	go func() {
		for range stderr {
			// consume stderr
		}
	}()

	go func() {
		for range stdout {
			// consume stdout
		}
	}()

	// Cancel after a short time
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for process to be canceled
	state, err := proc.Wait()
	if err == nil && state != nil && state.ExitCode() == 0 {
		t.Log("Process completed normally (may be too fast to cancel)")
	} else {
		t.Log("Process was canceled or failed as expected")
	}
}

func TestProcessState(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo test")
	} else {
		proc = New("echo", "test")
	}

	if proc.IsRunning() {
		t.Error("Process should not be running before Run()")
	}

	if proc.IsPaused() {
		t.Error("Process should not be paused before Run()")
	}

	// Don't call PID() before the process is started as it may panic
	// This is expected behavior - PID should only be called after Run()
}

func TestKillBeforeRun(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo test")
	} else {
		proc = New("echo", "test")
	}

	err := proc.Kill()
	if err == nil {
		t.Error("Kill() should fail on non-running process")
	}
}

func TestTerminateBeforeRun(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo test")
	} else {
		proc = New("echo", "test")
	}

	err := proc.Terminate()
	if err == nil {
		t.Error("Terminate() should fail on non-running process")
	}
}

func TestPauseResumeBeforeRun(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo test")
	} else {
		proc = New("echo", "test")
	}

	err := proc.Pause()
	if err == nil {
		t.Error("Pause() should fail on non-running process")
	}

	err = proc.Resume()
	if err == nil {
		t.Error("Resume() should fail on non-running process")
	}
}

// Benchmark tests
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var proc *Process
		if runtime.GOOS == windowsOS {
			proc = New("cmd", "/c", "echo benchmark")
		} else {
			proc = New("echo", "benchmark")
		}
		_ = proc
	}
}

func BenchmarkRunEcho(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var proc *Process
		if runtime.GOOS == windowsOS {
			proc = New("cmd", "/c", "echo benchmark test")
		} else {
			proc = New("echo", "benchmark test")
		}

		stdout, stderr, err := proc.Run()
		if err != nil {
			b.Fatalf("Run failed: %v", err)
		}

		go func() {
			for range stderr {
			}
		}()

		for range stdout {
		}

		_, _ = proc.Wait() // Ignore error and result in benchmark
	}
}

// Test PID functionality
func TestPID(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("ping", "-n", "3", "localhost")
	} else {
		proc = New("ping", "-c", "3", "localhost")
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Consume output
	go func() {
		for range stderr {
		}
	}()
	go func() {
		for range stdout {
		}
	}()

	// Test PID while process is running
	pid := proc.PID()
	if pid <= 0 {
		t.Errorf("Expected positive PID, got %d", pid)
	}

	// Wait for process to complete
	_, _ = proc.Wait()
}

// Test Write functionality
func TestWrite(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "findstr", "test")
	} else {
		proc = New("grep", "test")
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Write some input
	data := []byte("this is a test line\n")
	err = proc.Write(data)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Close stdin to signal end of input
	if proc.stdin != nil {
		_ = proc.stdin.Close()
	}

	// Consume output
	go func() {
		for range stderr {
		}
	}()

	var output []string
	for line := range stdout {
		output = append(output, line)
	}

	// Should find our test line
	if len(output) == 0 {
		t.Error("Expected output from grep/findstr, got none")
	}

	_, _ = proc.Wait()
}

// Test WriteString functionality
func TestWriteString(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "findstr", "hello")
	} else {
		proc = New("grep", "hello")
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Write string input
	testString := "hello world\n"
	err = proc.WriteString(testString)
	if err != nil {
		t.Fatalf("WriteString() failed: %v", err)
	}

	// Close stdin
	if proc.stdin != nil {
		_ = proc.stdin.Close()
	}

	// Consume output
	go func() {
		for range stderr {
		}
	}()

	var output []string
	for line := range stdout {
		output = append(output, line)
	}

	// Should find our hello line
	if len(output) == 0 {
		t.Error("Expected output from grep/findstr, got none")
	}

	_, _ = proc.Wait()
}

// Test KillWithTimeout functionality
func TestKillWithTimeout(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("ping", "-t", "localhost")
	} else {
		proc = New("yes") // Continuously outputs 'y'
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Start consuming output
	go func() {
		for range stderr {
		}
	}()
	go func() {
		for range stdout {
		}
	}()

	// Give process time to start
	time.Sleep(100 * time.Millisecond)

	// Test KillWithTimeout
	err = proc.KillWithTimeout(testTimeout * time.Second)
	if err != nil {
		t.Fatalf("KillWithTimeout() failed: %v", err)
	}

	// Process should no longer be running
	if proc.IsRunning() {
		t.Error("Process should not be running after KillWithTimeout")
	}
}

// Test Pause and Resume functionality (only on supported platforms)
func TestPauseResume(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("ping", "-n", "10", "localhost")
	} else {
		proc = New("ping", "-c", "10", "localhost")
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Consume output
	go func() {
		for range stderr {
		}
	}()
	go func() {
		for range stdout {
		}
	}()

	// Give process time to start
	time.Sleep(100 * time.Millisecond)

	// Test Pause
	err = proc.Pause()
	if err != nil {
		t.Logf("Pause() failed (may not be supported): %v", err)
	} else {
		if !proc.IsPaused() {
			t.Error("Process should be paused after Pause()")
		}

		// Test Resume
		time.Sleep(100 * time.Millisecond)
		err = proc.Resume()
		if err != nil {
			t.Errorf("Resume() failed: %v", err)
		} else if proc.IsPaused() {
			t.Error("Process should not be paused after Resume()")
		}
	}

	// Clean up
	_ = proc.Kill()
	_, _ = proc.Wait()
}

// Test error conditions for Write operations
func TestWriteErrors(t *testing.T) {
	proc := New("echo", "test")

	// Test writing to non-started process
	err := proc.Write([]byte("test"))
	if err == nil {
		t.Error("Write() should fail on non-running process")
	}

	err = proc.WriteString("test")
	if err == nil {
		t.Error("WriteString() should fail on non-running process")
	}
}

// Test Wait with already completed process
func TestWaitCompleted(t *testing.T) {
	var proc *Process
	if runtime.GOOS == windowsOS {
		proc = New("cmd", "/c", "echo done")
	} else {
		proc = New("echo", "done")
	}

	stdout, stderr, err := proc.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Consume output to allow process to complete
	go func() {
		for range stderr {
		}
	}()
	go func() {
		for range stdout {
		}
	}()

	// Give the process time to complete
	time.Sleep(100 * time.Millisecond)

	// Test that Wait() returns an error when process is already completed
	_, err1 := proc.Wait()
	if err1 == nil {
		t.Error("Expected Wait() to fail on completed process")
	}
	if !strings.Contains(err1.Error(), "process is not running") {
		t.Errorf("Expected 'process is not running' error, got: %v", err1)
	}

	// Test that the process is no longer running
	if proc.IsRunning() {
		t.Error("Process should not be running after completion")
	}
}
