# Copilot Instructions for fortio/log

## Project Overview

This is **Fortio's logging library** - a simple, opinionated logger built on top of Go's standard logger with additional log levels similar to glog but simpler to use and configure. The library provides both structured (JSON) and colorized console logging with goroutine ID tracking.

## Key Features

- **Multi-level logging**: Debug, Verbose, Info, Warning, Error, Critical, Fatal
- **Structured logging**: JSON output with fixed, optimized format
- **Console color output**: ANSI color codes for terminal output
- **HTTP request/response logging**: Built-in support for logging HTTP traffic
- **Goroutine ID tracking**: Track which goroutine generated log messages
- **Build tags for size optimization**: `no_http`, `no_net`, `no_json` to reduce binary size

## Project Structure

- `logger.go` - Core logging functionality and configuration
- `console_logging.go` - Colorized console output support
- `json_logging.go` - Structured JSON logging
- `no_json_logging.go` - Minimal logging without JSON marshaling support
- `http_logging.go` - HTTP request/response logging utilities
- `goroutine/` - Goroutine ID extraction utilities
- `levelsDemo/` - Example application demonstrating log levels

## Coding Standards

### Go Version
- Minimum: Go 1.18
- Use `GOTOOLCHAIN=local` for builds

### Code Style
- Follow standard Go conventions
- Use `golangci-lint` for linting (config fetched from fortio/workflows)
- Avoid global state mutation in tests
- Prefer table-driven tests

### Build Tags
The project supports conditional compilation using build tags:
- `no_http` - Exclude HTTP logging (reduces dependencies)
- `no_net` - Exclude all network-related functionality
- `no_json` - Exclude JSON marshaling for complex types (smallest binaries)

Always test changes with relevant build tags:
```bash
go test ./...
go test -tags no_json ./...
go test -tags no_http ./...
```

## Build & Test Commands

```bash
# Run all tests with race detection
make test

# Run tests with coverage
make coverage

# Lint the code
make lint

# Check binary sizes with different build tags
make size-check

# Run example demonstrating log levels
make example

# Run all checks (tests + lint + coverage + size check)
make all
```

## Important Patterns

### Log Configuration
- Use `Config` struct for all configuration options
- Environment variables use `LOGGER_*` prefix (e.g., `LOGGER_LOG_LEVEL`)
- Configuration is read from environment via `fortio.org/struct2env`

### Structured Logging
- Use `log.S()` for structured logging with attributes
- Attributes are key-value pairs: `log.Attr("key", value)`
- JSON output has fixed field order for performance: `ts`, `level`, `r` (goroutine), `file`, `line`, `msg`, then custom attributes

### Console vs JSON Output
- JSON output when stderr is redirected or `Config.JSON = true`
- Colorized output when terminal is detected and `Config.ConsoleColor = true`
- Colors defined in `log.Colors` map (empty strings when not in color mode)

### Testing
- Use `t.Parallel()` for test parallelization
- Reset log level with `defer log.SetLogLevel(prev)` to avoid test interference
- Mock stderr for testing output using `log.SetOutput()`

## Dependencies

- `fortio.org/struct2env` - Environment variable to struct configuration
- `github.com/kortschak/goroutine` - Goroutine ID extraction

## Special Considerations

1. **Goroutine ID Extraction**: Uses CGO-free method via stack trace parsing
2. **Timestamp Format**: Unix timestamp split into seconds.microseconds
3. **No Reflection in Hot Path**: JSON encoding optimized without reflection for performance
4. **Build Size Optimization**: HTTP logging pulls in many dependencies; use build tags to exclude if not needed

## Common Tasks

### Adding a New Log Level
1. Update the `Level` type and constants in `logger.go`
2. Add corresponding `*f()` function (e.g., `Debugf`, `Infof`)
3. Add JSON level string mapping
4. Update tests in `logger_test.go`
5. Update documentation in `README.md`

### Adding HTTP Logging Features
- Modify `http_logging.go`
- Ensure code is wrapped with `//go:build !no_http` constraint
- Add corresponding tests in `http_logging_test.go`
- Test with `go test -tags no_http` to ensure proper exclusion

### Modifying JSON Output Format
- Be careful: format is documented and consumed by tooling (e.g., `fortio.org/logc`)
- Update both `json_logging.go` and `no_json_logging.go` if changing base format
- Update JSONEntry struct if adding new base fields
- Consider backward compatibility

## Testing Philosophy

- Tests should pass with and without race detection
- Tests should pass with all supported build tag combinations
- Don't rely on specific log output format in tests (it may change)
- Test behavior, not implementation details
- Use table-driven tests for multiple similar test cases
