# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial release of processctrl package
- Cross-platform process management (Windows, Linux, macOS)
- Process pause/resume functionality
- Context support for cancellation and timeouts
- Buffered channels for high-throughput processes
- Graceful and forceful process termination
- Thread-safe process state management
- Real-time stdout/stderr streaming
- Interactive process communication via stdin
- Comprehensive example application
- Complete test suite
- CI/CD pipeline with GitHub Actions
- Cross-platform compatibility testing

### Features

- Start external processes with arguments
- Read live output from stdout and stderr using Go channels
- Write to process stdin for interactive processes
- Pause and resume processes (Linux/macOS: SIGSTOP/SIGCONT, Windows: NT APIs)
- Kill processes cleanly with graceful termination support
- Process state queries (IsRunning, IsPaused, PID)
- Wait for process completion with exit status
- Context-based cancellation and timeouts
- Configurable buffered channels

## [1.0.0] - 2025-08-01

### Added

- Initial stable release
