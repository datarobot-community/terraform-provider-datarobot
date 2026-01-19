# AGENTS.md - Terraform Provider DataRobot

## Project Overview
This is the official Terraform Provider for DataRobot, enabling infrastructure-as-code management of DataRobot resources.

## Core Commands

```bash
# Build and install
make build
make install

# Linting
make lint                    # Uses golangci-lint

# Unit tests
make test

# Acceptance tests (requires TF_ACC=1)
make testacc

# Generate docs
make generate

# Generate mocks
make mocks
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
- Unit tests: `go test ./... -v`
- Acceptance tests require DataRobot credentials and `TF_ACC=1`
- Mocks are generated with mockgen

### Code Style
- Go standard formatting (gofmt)
- Linting via golangci-lint
- Follow existing patterns in the codebase

## Git Workflow
- Main branch: `main`
- Create feature branches for changes
- Update CHANGELOG.md for user-facing changes
