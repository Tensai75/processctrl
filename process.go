// Package processctrl provides cross-platform process management capabilities
// for Go applications. It allows you to start, control, and monitor external
// processes with features like:
//
//   - Real-time stdout/stderr streaming via Go channels
//   - Process pause/resume functionality (platform-specific)
//   - Graceful and forceful process termination
//   - Context-based cancellation and timeouts
//   - Interactive process communication via stdin
//   - Thread-safe process state management
//
// The package abstracts platform differences between Unix-like systems
// (Linux, macOS) and Windows, providing a consistent API while using
// the most appropriate underlying mechanisms for each platform.
//
// Example usage:
//
//	proc := processctrl.New("ping", "localhost")
//	stdout, stderr, err := proc.Run()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	go func() {
//		for line := range stdout {
//			fmt.Println("OUT:", line)
//		}
//	}()
//
//	time.Sleep(5 * time.Second)
//	proc.Kill()
package processctrl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	// defaultKillTimeout is the default timeout for graceful termination
	defaultKillTimeout = 5 * time.Second
	// streamGoroutines is the number of goroutines used for streaming output
	streamGoroutines = 2
)

// Process represents a managed external process with controllable execution.
// It provides channels for reading stdout/stderr output and supports
// pause/resume functionality across different platforms.
type Process struct {
	program        string
	args           []string
	cmd            *exec.Cmd
	stdout         chan string
	stderr         chan string
	stdin          io.WriteCloser
	mu             sync.RWMutex
	paused         bool
	running        bool
	bufferSize     int
	channelsClosed bool
}

// New creates a new Process instance with unbuffered output channels.
// The process is not started until Run() or RunWithContext() is called.
//
// Parameters:
//   - program: The executable program to run
//   - args: Command line arguments to pass to the program
//
// Returns a new Process instance ready to be started.
func New(program string, args ...string) *Process {
	return NewWithBuffer(0, program, args...)
}

// NewWithBuffer creates a new Process instance with buffered output channels.
// Buffered channels can improve performance for processes that produce high volumes
// of output by reducing blocking between the process and the consumer.
//
// Parameters:
//   - bufferSize: Size of the buffer for stdout/stderr channels (0 for unbuffered)
//   - program: The executable program to run
//   - args: Command line arguments to pass to the program
//
// Returns a new Process instance ready to be started.
func NewWithBuffer(bufferSize int, program string, args ...string) *Process {
	return &Process{
		program:        program,
		args:           args,
		stdout:         make(chan string, bufferSize),
		stderr:         make(chan string, bufferSize),
		bufferSize:     bufferSize,
		channelsClosed: false,
		// cmd will be created in RunWithContext with proper context
	}
}

// Run starts the process with a background context and returns channels for
// reading stdout and stderr output. This is equivalent to calling
// RunWithContext(context.Background()).
//
// Returns:
//   - stdout channel: Receives lines from the process's standard output
//   - stderr channel: Receives lines from the process's standard error
//   - error: Any error that occurred during process startup
//
// The channels will be closed when the process exits or is terminated.
func (p *Process) Run() (<-chan string, <-chan string, error) {
	return p.RunWithContext(context.Background())
}

// RunWithContext starts the process with the given context and returns channels
// for reading stdout and stderr output. The context can be used for cancellation
// and timeouts - when the context is canceled, the process will be terminated.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - stdout channel: Receives lines from the process's standard output
//   - stderr channel: Receives lines from the process's standard error
//   - error: Any error that occurred during process startup
//
// The process will be automatically terminated if the context is canceled.
// The channels will be closed when the process exits or is terminated.
func (p *Process) RunWithContext(ctx context.Context) (<-chan string, <-chan string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil, nil, fmt.Errorf("process already running")
	}

	// Create command with context for proper cancellation
	p.cmd = exec.CommandContext(ctx, p.program, p.args...)

	stdoutPipe, err := p.cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := p.cmd.StderrPipe()
	if err != nil {
		_ = stdoutPipe.Close() // Ignore error during cleanup
		return nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	stdinPipe, err := p.cmd.StdinPipe()
	if err != nil {
		_ = stdoutPipe.Close() // Ignore error during cleanup
		_ = stderrPipe.Close() // Ignore error during cleanup
		return nil, nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	p.stdin = stdinPipe

	if err := p.cmd.Start(); err != nil {
		_ = stdoutPipe.Close() // Ignore error during cleanup
		_ = stderrPipe.Close() // Ignore error during cleanup
		_ = p.stdin.Close()    // Ignore error during cleanup
		return nil, nil, fmt.Errorf("failed to start process: %w", err)
	}

	p.running = true

	var wg sync.WaitGroup
	wg.Add(streamGoroutines)

	go streamOutput(stdoutPipe, p.stdout, &wg)
	go streamOutput(stderrPipe, p.stderr, &wg)

	go func() {
		defer func() {
			p.mu.Lock()
			p.running = false
			p.paused = false
			if p.stdin != nil {
				_ = p.stdin.Close() // Ignore error during cleanup
			}
			p.mu.Unlock()

			// Close channels outside the mutex to avoid holding lock too long
			p.closeChannels()
		}()

		// Wait for either process completion or context cancellation
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Process completed normally
		case <-ctx.Done():
			// Context was canceled
			_ = p.cmd.Process.Kill() // Ignore error as process might already be dead
			<-done                   // Wait for streams to finish
		}
	}()

	return p.stdout, p.stderr, nil
}

// streamOutput reads from an io.Reader and sends each line to a channel.
// This function is used internally to stream stdout and stderr output
// from the process to the respective channels.
//
// Parameters:
//   - r: The reader to read from (typically stdout or stderr pipe)
//   - ch: The channel to send lines to
//   - wg: WaitGroup to signal completion
func streamOutput(r io.Reader, ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
}

// closeChannels safely closes the stdout and stderr channels.
// This method ensures channels are only closed once by tracking
// the closure state to prevent panics from double-closing.
func (p *Process) closeChannels() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.channelsClosed {
		close(p.stdout)
		close(p.stderr)
		p.channelsClosed = true
	}
}

// Kill forcefully terminates the process without graceful shutdown.
// This is an immediate termination that may cause data loss.
// For graceful termination, use Terminate() instead.
//
// Returns an error if the process is not running or termination fails.
func (p *Process) Kill() error {
	return p.killWithSignal(defaultKillTimeout, true)
}

// KillWithTimeout attempts to terminate the process with a custom timeout.
// This method uses the platform-specific killWithSignal implementation
// to force-kill the process after the specified timeout.
//
// Parameters:
//   - timeout: Maximum time to wait before force-killing the process
//
// Returns an error if the operation fails.
func (p *Process) KillWithTimeout(timeout time.Duration) error {
	return p.killWithSignal(timeout, false)
}

// Terminate attempts to gracefully stop the process by first sending a
// termination signal, then forcing termination if the process doesn't exit
// within a reasonable timeout period.
//
// This method provides a balance between allowing graceful shutdown and ensuring
// the process is eventually terminated.
//
// Returns an error if the operation fails.
func (p *Process) Terminate() error {
	return p.killWithSignal(defaultKillTimeout, true)
}

// Wait blocks until the process completes and returns its exit status.
// This method should be called after starting a process to get the final
// exit code and ensure proper cleanup.
//
// Returns:
//   - *os.ProcessState: Contains exit code and other process completion info
//   - error: If the process is not running or if waiting fails
//
// Note: This method will block until the process exits naturally or is killed.
func (p *Process) Wait() (*os.ProcessState, error) {
	p.mu.RLock()
	cmd := p.cmd
	running := p.running
	p.mu.RUnlock()

	if !running {
		return nil, fmt.Errorf("process is not running")
	}

	return cmd.ProcessState, cmd.Wait()
}

// IsRunning returns true if the process is currently running.
// This method is thread-safe and can be called concurrently.
//
// Returns true if the process has been started and has not yet exited.
func (p *Process) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// IsPaused returns true if the process is currently paused.
// A paused process is still running but temporarily suspended.
// This method is thread-safe and can be called concurrently.
//
// Returns true if the process is currently in a paused state.
func (p *Process) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paused
}

// PID returns the process ID of the running process.
// This method is thread-safe and can be called concurrently.
//
// Returns:
//   - Process ID (positive integer) if the process is running
//   - -1 if the process has not been started or has exited
func (p *Process) PID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return -1
}

// Write sends data to the process's standard input.
// This method allows interaction with processes that read from stdin.
// The method is thread-safe and can be called concurrently.
//
// Parameters:
//   - data: Byte slice to send to the process's stdin
//
// Returns an error if:
//   - The process is not running
//   - Stdin is not available
//   - The write operation fails
func (p *Process) Write(data []byte) error {
	p.mu.RLock()
	stdin := p.stdin
	running := p.running
	p.mu.RUnlock()

	if !running {
		return fmt.Errorf("process is not running")
	}

	if stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	_, err := stdin.Write(data)
	return err
}

// WriteString is a convenience method that sends a string to the process's
// standard input. This is equivalent to calling Write([]byte(s)).
//
// Parameters:
//   - s: String to send to the process's stdin
//
// Returns an error if the write operation fails (see Write for details).
func (p *Process) WriteString(s string) error {
	return p.Write([]byte(s))
}
