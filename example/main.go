// Example program demonstrating the processctrl package features.
// This example shows how to:
//   - Create a process with buffered channels for better performance
//   - Use context for automatic timeout and cancellation
//   - Monitor process state (running, paused, PID)
//   - Control process execution (pause, resume, terminate)
//   - Handle process output through channels
//   - Wait for process completion and get exit status
package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/tensai75/processctrl"
)

func main() {
	// Create a process with buffered channels (buffer size: 10)
	// Buffered channels help with high-output processes
	// Use appropriate ping flags for continuous pinging on different platforms
	var proc *processctrl.Process
	if runtime.GOOS == "windows" {
		// On Windows, use -t flag to ping continuously until stopped
		proc = processctrl.NewWithBuffer(10, "ping", "-t", "localhost")
	} else {
		// On Unix systems (Linux/macOS), ping runs continuously by default
		// but we can use -i to set interval to make it more responsive
		proc = processctrl.NewWithBuffer(10, "ping", "-i", "1", "localhost")
	}

	// Run with context and timeout for automatic cancellation
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

	time.Sleep(3 * time.Second)
	fmt.Println("Pausing...")
	if err := proc.Pause(); err != nil {
		fmt.Printf("Failed to pause: %v\n", err)
	} else {
		fmt.Printf("Process paused: %v\n", proc.IsPaused())
	}

	time.Sleep(5 * time.Second)
	fmt.Println("Resuming...")
	if err := proc.Resume(); err != nil {
		fmt.Printf("Failed to resume: %v\n", err)
	} else {
		fmt.Printf("Process paused: %v\n", proc.IsPaused())
	}

	time.Sleep(5 * time.Second)
	fmt.Println("Terminating gracefully...")
	if err := proc.Terminate(); err != nil {
		fmt.Printf("Failed to terminate gracefully: %v\n", err)
		fmt.Println("Force killing...")
		proc.Kill()
	}

	// Wait for process to complete
	if state, err := proc.Wait(); err == nil && state != nil {
		fmt.Printf("Process exited with code: %d\n", state.ExitCode())
	}
}
