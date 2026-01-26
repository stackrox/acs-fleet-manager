# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ACS Fleet Manager is a Red Hat Advanced Cluster Security (ACS) managed service that allows users to request and manage ACS instances through the Red Hat Cloud Console. The service consists of two main components:

- **Fleet Manager**: Control plane service that accepts user requests and manages ACS instance provisioning
- **Fleetshard-sync**: Data plane component that provisions ACS instances on OpenShift clusters

## Common Commands

### Build and Development
```bash
# Build all binaries and generate OpenAPI
make all

# Build only binaries
make binary

# Generate code (OpenAPI, mocks, etc.)
make generate

# Clean build artifacts
make clean
```

### Testing
```bash
# Run unit tests
make test

# Run integration tests
make test/integration

# Run E2E tests
make test/e2e

# Run specific test with flags
make test TESTFLAGS="-run TestSomething"

# Run AWS integration tests
make test/aws
```

### Database Operations
```bash
# Setup local database
make db/setup

# Run database migrations
make db/migrate

# Teardown local database
make db/teardown

# Access database CLI
make db/login
```

### Code Quality
```bash
# Lint code
make lint

# Verify code (includes vet and OpenAPI validation)
make verify

# Check code formatting
make code/check

# Fix code formatting
make code/fix

# Validate OpenAPI specifications
make openapi/validate
```

### Development Environment
```bash
# Bootstrap development environment
make deploy/bootstrap

# Deploy to development cluster
make deploy/dev

# Fast redeploy for development
make deploy/dev-fast

# Reset E2E test environment
make test/e2e/reset
```

### Container Images
```bash
# Build container image
make image/build

# Push image to registry
make image/push

# Build specific component images
make image/build/probe
make image/build/emailsender
```

## Architecture

### Core Components

1. **Fleet Manager** (`cmd/fleet-manager/`) - Main control plane service
2. **Fleetshard-sync** (`fleetshard/`) - Data plane synchronization service
3. **Probe** (`probe/`) - Health check and monitoring service
4. **Emailsender** (`emailsender/`) - Email notification service

### Key Directories

- `internal/central/` - Central service implementation (main business logic)
- `pkg/` - Shared packages and utilities
- `openapi/` - OpenAPI specifications for all APIs
- `docs/` - Documentation including architecture and development guides
- `templates/` - Kubernetes/OpenShift deployment templates
- `dev/env/` - Development environment configuration
- `e2e/` - End-to-end test suites

### Service Architecture

The codebase follows a dependency injection pattern using the `goava/di` framework:

- `internal/central/providers.go` - Main dependency injection configuration
- Services are organized into:
  - **Services**: Business logic (`internal/central/pkg/services/`)
  - **Workers**: Background reconciliation (`internal/central/pkg/workers/`)
  - **Handlers**: API endpoints (`internal/central/pkg/handlers/`)
  - **Presenters**: Data transformation (`internal/central/pkg/presenters/`)

### Database

- Uses PostgreSQL with GORM ORM
- Database migrations in `internal/central/pkg/migrations/`
- Local development uses Docker container

### Testing Strategy

- Unit tests: Standard Go testing with testify
- Integration tests: Test against real database and external services
- E2E tests: Use Ginkgo framework for comprehensive end-to-end scenarios
- AWS integration tests: Test cloud provider integrations

## Development Workflow

### Environment Setup
1. Install prerequisites: Go 1.25+, Docker, Node.js, Java, OCM CLI
2. Run `make setup/git/hooks` to install pre-commit hooks
3. Use `make deploy/bootstrap` to set up development cluster
4. Run `make deploy/dev` to start local development environment

### Code Generation
- OpenAPI client/server code is generated - never edit generated files directly
- Run `make generate` after modifying OpenAPI specs or adding `//go:generate` directives
- Generated files are in `internal/central/pkg/api/` and other `*_moq.go` files

### Pre-commit Hooks
- golangci-lint for Go code quality
- shellcheck for shell scripts
- detect-secrets for security scanning
- Generated files verification
- Code formatting checks

### Configuration
- Environment-specific configs in `dev/config/`
- Secret files in `secrets/` directory (use `make secrets/touch` to create empty files)
- Multiple cluster types supported: minikube, kind, colima, CRC, OpenShift

## Important Notes

- Always run `make lint` and `make verify` before committing
- Database migrations are applied automatically on startup
- The service uses Red Hat SSO for authentication
- Multiple API specs: public, private, and admin APIs
- Container images are built for linux/amd64 platform
- Use `./scripts/fmcurl` for API testing
- Fleet manager stores state in PostgreSQL, fleetshard-sync is stateless
