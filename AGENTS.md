# AGENTS.md - Terraform Provider DataRobot

## Project Overview
This is the official Terraform Provider for DataRobot, enabling infrastructure-as-code management of DataRobot resources.

## Core Commands

```bash
# Build and install
task build
task install

# Linting
task lint                    # Uses golangci-lint

# Security vulnerability scan
task vuln                    # Uses govulncheck (Go CVE database)

# Unit tests
task test

# Unit tests with coverage report
task test-coverage

# Acceptance tests (requires TF_ACC=1)
task testacc

# Generate docs
task generate

# Generate mocks
task mocks
```

## Project Structure
- `internal/` - Provider implementation (resources, data sources)
- `pkg/` - Shared packages
- `docs/` - Terraform documentation
- `examples/` - Example Terraform configurations
- `mock/` - Mock implementations for testing
- `test/` - Additional test utilities

## Development Patterns

### Resource Implementation
- Resources are in `internal/`
- Follow Terraform Plugin Framework patterns
- Use the client service interface from `internal/client/service.go`

### Testing
- Unit tests: `task test` or `go test ./... -v`
- Unit tests with coverage: `task test-coverage` (generates coverage.out report)
- Acceptance tests require DataRobot credentials and `TF_ACC=1`
- CI uses `gotestsum` for test timing, JUnit reports, and flaky test retries (--rerun-fails)
- Test results uploaded as artifacts (test-results.xml, coverage.out)
- Mocks are generated with mockgen

### Code Style
- Go standard formatting (gofmt)
- Linting via golangci-lint
- Follow existing patterns in the codebase

## Git Workflow
- Main branch: `main`
- Create feature branches for changes
- Update CHANGELOG.md for user-facing changes
