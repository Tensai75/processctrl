# processctrl

`processctrl` is a cross-platform Go package for managing external processes.
It allows you to run a subprocess, read from its stdout and stderr via channels, pause/resume its execution, and kill it at any time.

## Features

- ✅ Start external processes with arguments
- ✅ Read live output from stdout and stderr using Go channels
- ✅ Write to process stdin for interactive processes
- ✅ Pause and resume processes
  - Linux/macOS: via SIGSTOP/SIGCONT
  - Windows: via NtSuspendProcess/NtResumeProcess from `ntdll.dll`
- ✅ Kill processes cleanly with graceful termination support
- ✅ Thread-safe state management with RWMutex and flags
- ✅ Context support for cancellation and timeouts
- ✅ Configurable buffered channels for high-throughput processes
- ✅ Process state queries (IsRunning, IsPaused, PID)
- ✅ Wait for process completion with exit status
- ✅ Improved error messages with context

## Installation

```
go get github.com/tensai75/processctrl
```

## Example Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/tensai75/processctrl"
)

func main() {
    // Create a process with buffered channels (buffer size 10)
    proc := processctrl.NewWithBuffer(10, "ping", "localhost")

    // Run with context and timeout
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    stdout, stderr, err := proc.RunWithContext(ctx)
    if err != nil {
        panic(err)
    }

    // Check process state
    fmt.Printf("Process running: %v, PID: %d\n", proc.IsRunning(), proc.PID())

    go func() {
        for line := range stdout {
            fmt.Println("STDOUT:", line)
        }
    }()

    go func() {
        for line := range stderr {
            fmt.Println("STDERR:", line)
        }
    }()

    // For interactive processes, you can write to stdin
    // proc.WriteString("some input\n")

    time.Sleep(3 * time.Second)
    fmt.Println("Pausing...")
    proc.Pause()

    time.Sleep(5 * time.Second)
    fmt.Println("Resuming...")
    proc.Resume()

    time.Sleep(5 * time.Second)

    // Try graceful termination first (SIGTERM on Unix, gentle close on Windows)
    fmt.Println("Terminating gracefully...")
    if err := proc.Terminate(); err != nil {
        fmt.Println("Graceful termination failed, force killing...")
        proc.Kill()
    }

    // Wait for process to complete and get exit status
    if state, err := proc.Wait(); err == nil && state != nil {
        fmt.Printf("Process exited with code: %d\n", state.ExitCode())
    }
}
```

## API Reference

### Creating Processes

```go
// Create process with unbuffered channels
proc := processctrl.New("command", "arg1", "arg2")

// Create process with buffered channels (recommended for high-output processes)
proc := processctrl.NewWithBuffer(100, "command", "arg1", "arg2")
```

### Running Processes

```go
// Run with default context
stdout, stderr, err := proc.Run()

// Run with custom context (supports cancellation and timeouts)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
stdout, stderr, err := proc.RunWithContext(ctx)
```

### Process Control

```go
// Pause/Resume (platform-specific implementation)
err := proc.Pause()
err := proc.Resume()

// Termination options
err := proc.Kill()                              // Force kill immediately
err := proc.KillWithTimeout(5 * time.Second)    // Force kill with timeout
err := proc.Terminate()                         // Graceful termination (5s timeout)
```

### Process State

```go
running := proc.IsRunning()  // Check if process is running
paused := proc.IsPaused()    // Check if process is paused
pid := proc.PID()            // Get process ID (-1 if not running)
```

### Input/Output

```go
// Write to process stdin
err := proc.Write([]byte("input data"))
err := proc.WriteString("input string\n")

// Wait for completion and get exit status
state, err := proc.Wait()
if err == nil && state != nil {
    exitCode := state.ExitCode()
}
```

## Platform Compatibility

| Platform    | Pause/Resume Method                                   |
| ----------- | ----------------------------------------------------- |
| Linux/macOS | POSIX signals (`SIGSTOP`/`SIGCONT`)                   |
| Windows     | `NtSuspendProcess`/`NtResumeProcess` from `ntdll.dll` |

## License

MIT
