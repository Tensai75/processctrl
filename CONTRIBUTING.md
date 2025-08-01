# Contributing to processctrl

Thank you for your interest in contributing to processctrl! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to a code of conduct that we expect all participants to follow. Please be respectful and professional in all interactions.

## How to Contribute

### Reporting Issues

- Use the GitHub issue tracker to report bugs or request features
- Before creating an issue, please search existing issues to avoid duplicates
- Include as much detail as possible:
  - Go version and operating system
  - Steps to reproduce the issue
  - Expected vs actual behavior
  - Code examples if applicable

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following the coding standards below
3. **Add tests** for your changes if applicable
4. **Run the test suite** to ensure all tests pass
5. **Update documentation** if you've changed APIs
6. **Submit a pull request** with a clear description of your changes

## Development Setup

1. Clone your fork:

   ```bash
   git clone https://github.com/YOUR_USERNAME/processctrl.git
   cd processctrl
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Install development tools:

   ```bash
   make install-tools
   ```

4. Run tests to ensure everything works:
   ```bash
   make test
   ```

## Coding Standards

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Run `golangci-lint` to check for issues
- Use meaningful variable and function names
- Write clear, concise comments

### Testing

- Write tests for new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Test cross-platform compatibility when relevant

### Documentation

- Update README.md if you change functionality
- Add inline comments for complex logic
- Update CHANGELOG.md for notable changes
- Include examples in documentation

## Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make bench

# Run linter
make lint
```

## Platform-Specific Considerations

This package supports Windows, Linux, and macOS. When making changes:

- Test on multiple platforms when possible
- Use build tags (`//go:build`) for platform-specific code
- Consider platform differences in process management
- Update platform-specific files (`process_windows.go`, `process_unix.go`) as needed

## Commit Messages

Use clear, descriptive commit messages:

```
Add context support for process cancellation

- Implement RunWithContext method
- Add context cancellation in process execution
- Update tests to cover context scenarios
- Update documentation with context examples
```

## Pull Request Process

1. Ensure your PR addresses a single concern
2. Include tests for new functionality
3. Update documentation as needed
4. Ensure CI passes
5. Respond to review feedback promptly

## Release Process

Releases follow semantic versioning:

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Getting Help

- Open an issue for questions about contributing
- Check existing issues and documentation first
- Be patient and respectful when asking for help

## Recognition

Contributors will be recognized in the project documentation and release notes.

Thank you for contributing to processctrl!
