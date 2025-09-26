# Design Proposal: Remove Dead Sentry Code

## Summary

This proposal outlines the removal of unused Sentry logging functionality from the ACS Fleet Manager codebase. The Sentry error monitoring feature was implemented but is not being used in production, making it dead code that should be cleaned up.

## Background

Sentry is an error monitoring service that was integrated into Fleet Manager to capture and report errors in real-time. However, this functionality is currently disabled and not used in production environments. The code, configuration, and documentation related to Sentry should be removed to:

- Reduce codebase complexity
- Remove unused dependencies
- Simplify configuration
- Eliminate dead code paths

## Current Sentry Implementation

### Core Package Files
The following files constitute the main Sentry implementation:

1. **`pkg/services/sentry/sentry.go`** - Main Sentry initialization logic
2. **`pkg/services/sentry/config.go`** - Sentry configuration structure and flags
3. **`pkg/services/sentry/providers.go`** - Dependency injection setup for Sentry

### Integration Points

1. **Logger Integration** (`pkg/logger/logger.go`):
   - Lines 10, 99, 108, 215, 223, 230-233, 240, 243-253: Sentry hub usage and event capture
   - Sentry integration in error, warning, and fatal logging methods

2. **Core Providers** (`pkg/providers/core.go`):
   - Line 23: Import of sentry services
   - Line 49: Sentry config providers registration

3. **Environment Configurations**:
   - `internal/central/pkg/environments/development.go` - Line 17: Sentry disabled
   - `internal/central/pkg/environments/integration.go` - References to Sentry
   - `internal/central/pkg/environments/production.go` - Sentry configuration

### Configuration and Deployment Files

1. **Kubernetes Templates**:
   - `templates/secrets-template.yml`:
     - Lines 53-54: SENTRY_KEY parameter definition
     - Line 103: sentry.key secret mapping
   - `templates/service-template.yml`:
     - Lines 158-178: Sentry-related parameters (ENABLE_SENTRY, SENTRY_URL, etc.)
     - Lines 663, 946-951: Sentry configuration flags in deployment

2. **Makefile**:
   - References to `secrets/sentry.key` file
   - Sentry key parameter handling in deployment targets

### Documentation References

1. **Feature Documentation**:
   - `docs/legacy/feature-flags.md` - Lines 18, 93-100: Sentry feature flag documentation
   - `docs/development/running-fleet-manager.md` - Lines 11, 19: Environment descriptions mentioning Sentry
   - `docs/development/populating-configuration.md` - Lines 118-132: Sentry configuration setup
   - `docs/legacy/development/error-handling.md` - Lines 61, 66, 68, 90: Error handling with Sentry
   - `CONTRIBUTING.md` - Lines 116-136: Sentry logging examples

### Dependencies

1. **Go Module** (`go.mod`):
   - Line 22: `github.com/getsentry/sentry-go v0.34.0` dependency

### Configuration Files

1. **Environment Files**:
   - Various environment files contain Sentry-related configurations that need to be cleaned up

## Removal Plan

### Phase 1: Remove Core Sentry Package
1. Delete the entire `pkg/services/sentry/` directory and all its contents:
   - `pkg/services/sentry/sentry.go`
   - `pkg/services/sentry/config.go`
   - `pkg/services/sentry/providers.go`

### Phase 2: Remove Integration Points
1. **Update `pkg/logger/logger.go`**:
   - Remove Sentry import (line 10)
   - Remove `sentryHub *sentry.Hub` field from logger struct (line 99)
   - Remove Sentry hub initialization in `NewUHCLogger` (line 108)
   - Remove `captureSentryEvent` calls from `Warningf`, `Errorf`, and `Fatalf` methods
   - Remove entire `captureSentryEvent` method (lines 243-253)
   - Simplify `Error` method to remove Sentry capture logic (lines 229-234)

2. **Update `pkg/providers/core.go`**:
   - Remove Sentry import (line 23)
   - Remove Sentry config providers call (line 49)

### Phase 3: Clean Up Configuration and Deployment
1. **Update Kubernetes templates**:
   - `templates/secrets-template.yml`:
     - Remove SENTRY_KEY parameter (lines 53-54)
     - Remove sentry.key from secrets (line 103)
   - `templates/service-template.yml`:
     - Remove all Sentry-related parameters (lines 158-178)
     - Remove Sentry command-line flags from deployment (lines 946-951)
     - Remove sentry.key from fake configmap (line 663)

2. **Update Makefile**:
   - Remove any Sentry key handling logic
   - Remove references to `secrets/sentry.key`

### Phase 4: Remove Environment Configurations
1. **Update environment files**:
   - `internal/central/pkg/environments/development.go` - Remove Sentry references
   - `internal/central/pkg/environments/integration.go` - Remove Sentry references  
   - `internal/central/pkg/environments/production.go` - Remove Sentry references

### Phase 5: Update Documentation
1. **Remove or update documentation files**:
   - `docs/legacy/feature-flags.md` - Remove Sentry section (lines 18, 93-100)
   - `docs/development/running-fleet-manager.md` - Remove Sentry mentions from environment descriptions
   - `docs/development/populating-configuration.md` - Remove Sentry configuration section (lines 118-132)
   - `docs/legacy/development/error-handling.md` - Remove Sentry references and update error handling examples
   - `CONTRIBUTING.md` - Remove Sentry logging examples and references

### Phase 6: Remove Dependencies
1. **Update `go.mod`**:
   - Remove `github.com/getsentry/sentry-go v0.34.0` dependency
   - Run `go mod tidy` to clean up unused dependencies

### Phase 7: Clean Up Configuration Files
1. **Remove Sentry configuration from environment config files**:
   - Check `dev/config/` directory for any Sentry-related configurations
   - Remove any `secrets/sentry.key` file references or creation scripts

## Testing Considerations

1. **Unit Tests**: Verify that logger tests still pass after removing Sentry integration
2. **Integration Tests**: Ensure that error handling still works correctly without Sentry
3. **Build Verification**: Confirm that the application builds and starts successfully after removal
4. **Deployment Tests**: Verify that Kubernetes deployments work without Sentry configuration

## Migration Notes

1. **No Data Migration Required**: Since Sentry is not currently used, no data migration is needed
2. **Configuration Updates**: Existing deployments will need configuration updates to remove Sentry-related parameters
3. **Error Monitoring**: If error monitoring is needed in the future, a different solution should be implemented

## Verification Steps

After implementing the removal:

1. **Code Verification**:
   - Search codebase for any remaining "sentry" references (case-insensitive)
   - Verify `go mod tidy` removes the Sentry dependency
   - Ensure all imports are valid and no unused imports remain

2. **Build Verification**:
   - Run `make binary` to ensure compilation succeeds
   - Run `make test` to verify tests pass
   - Run `make lint` to check for any linting issues

3. **Deployment Verification**:
   - Verify Kubernetes templates are valid after removing Sentry parameters
   - Test local deployment without Sentry configuration

## Benefits

1. **Reduced Complexity**: Removes unused code paths and configurations
2. **Cleaner Dependencies**: Eliminates unnecessary external dependency
3. **Simplified Configuration**: Reduces the number of configuration parameters
4. **Maintainability**: Less code to maintain and fewer potential security vulnerabilities
5. **Documentation Clarity**: Removes outdated documentation about unused features

## Risks

1. **Accidental Usage**: Low risk that some code might still reference Sentry (mitigated by comprehensive search)
2. **Future Needs**: If error monitoring is needed later, it will require re-implementation (acceptable trade-off for current cleanup)

## Estimated Effort

- **Development Time**: ~2-3 hours for a junior engineer
- **Testing Time**: ~1-2 hours
- **Documentation Update**: ~1 hour
- **Total**: ~4-6 hours

This removal should be straightforward since Sentry is not actively used, making it a low-risk cleanup operation.