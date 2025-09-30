# Configuration Cleanup Design Proposal

**Author**: River (Configuration Expert)  
**Date**: 2025-09-26  
**Status**: Draft

## Summary

This proposal outlines recommended cleanups for the ACS Fleet Manager configuration system based on analysis of vault configuration files and actual usage patterns in the codebase. The analysis identified several opportunities to reduce configuration complexity, remove unused fields, and eliminate unnecessary dummy values.

## Current State Analysis

### Vault Configuration Files Analyzed

1. **Integration Environment** (`.tmp/vault_configs/integration-secret.yaml`)
2. **Staging Environment** (`.tmp/vault_configs/stage-secret.json`)
3. **Production Environment** (`.tmp/vault_configs/prod-secret.json`)

### Configuration Loading Patterns

The Fleet Manager uses a file-based secret loading pattern where:
- Secret file paths are defined in configuration structs (e.g., `aws.AccessKeyFile`)
- Secret values are loaded at runtime into corresponding fields (e.g., `aws.AccessKey`)
- Configuration is managed through dependency injection with environment-specific defaults

## Identified Issues and Recommendations

### 1. Unused Secret Values (High Priority)

**Issue**: Several secrets are provided in vault but never referenced in the codebase.

#### Unused Keycloak Service Credentials
- **Found in vault**: `keycloak-service.clientId`, `keycloak-service.clientSecret`
- **Status**: No references found in Go codebase
- **Recommendation**: **Remove** these secrets from all environments

#### Unused OSD IDP Keycloak Service Credentials  
- **Found in vault**: `osd-idp-keycloak-service.clientId`, `osd-idp-keycloak-service.clientSecret`
- **Status**: No references found in Go codebase
- **Recommendation**: **Remove** these secrets from all environments

#### Unused Image Pull Configuration
- **Found in vault**: `image-pull.dockerconfigjson`
- **Status**: No references found in Go codebase
- **Recommendation**: **Remove** this secret from all environments

#### Unused Observability Configuration
- **Found in vault**: `observability-config-access.token`
- **Status**: No references found in Go codebase  
- **Recommendation**: **Remove** this secret from all environments

### 2. Dummy Values That Should Be Eliminated (Medium Priority)

**Issue**: Several secrets use dummy/placeholder values that indicate either unused functionality or hardcodable values.

#### AWS Credentials with Dummy Values
- **Found**: `aws.accesskey: "dummySecret"`, `aws.secretaccesskey: "dummyKey"` (all environments) <!-- pragma: allowlist secret -->
- **Analysis**: These are actively used in `internal/central/pkg/config/aws.go` for OSD cluster creation
- **Recommendation**:
  - **Investigation needed**: Determine if these are actually used or if OCM integration bypasses them
  - If unused: Remove AWS access key/secret fields entirely and rely only on Route53 credentials
  - If used: Ensure proper credentials are provided instead of dummy values

#### Central IDP Client Secret
- **Found**: `central.idp-client-secret: "dummySecret"` (integration/staging), real value in production <!-- pragma: allowlist secret -->
- **Analysis**: Used in `internal/central/pkg/config/central.go` for static authentication configuration
- **Recommendation**:
  - **Investigation needed**: Verify if static IDP configuration is still required
  - Consider if this can be moved to environment-specific defaults rather than secrets

#### Keycloak/Sentry Dummy Values
- **Found**: `keycloak-service.*: "dummyId/dummySecret"`, `sentry.key: "dummyKey"`
- **Status**: No usage found in codebase
- **Recommendation**: **Remove** entirely as they appear to be legacy configuration

### 3. Hardcodable Values (Low Priority)

**Issue**: Some configuration values are identical across all environments and could be hardcoded.

#### Red Hat SSO Client ID
- **Found**: `redhatsso-service.clientId: "rhacs-fleet-manager"` (identical across all environments)
- **Analysis**: Used in `pkg/client/iam/config.go` for Red Hat SSO integration
- **Recommendation**: **Hardcode** this value in the Go configuration struct and remove from secrets

#### Telemetry Configuration Pattern
- **Found**: Telemetry disabled in integration (`"DISABLED"`), enabled with keys in staging/production
- **Analysis**: The code already handles missing telemetry keys gracefully
- **Recommendation**: Consider making telemetry opt-in through environment flags rather than secrets

### 4. Environment-Specific Inconsistencies (Medium Priority)

#### OCM Service Client ID Variations
- **Integration**: `rhacs-fleetmanager-ocm-integration`
- **Staging**: `rhacs-fleetmanager-ocm`  
- **Production**: `rhacs-fleetmanager-ocm-prod`
- **Recommendation**: **Keep as-is** - these appear to be legitimate environment-specific service accounts

#### Account ID Patterns
- **Found**: Different AWS account IDs per environment (expected)
- **Recommendation**: **Keep as-is** - legitimate environment separation

## Implementation Plan

### Phase 1: Remove Unused Secrets (Immediate)
1. **Remove unused Keycloak service configurations**:
   - `keycloak-service.clientId`
   - `keycloak-service.clientSecret`
   - `osd-idp-keycloak-service.clientId`
   - `osd-idp-keycloak-service.clientSecret`

2. **Remove unused infrastructure secrets**:
   - `image-pull.dockerconfigjson`
   - `observability-config-access.token`
   - `sentry.key` (if confirmed unused)

### Phase 2: Investigate and Clean Dummy Values (Short-term)
1. **AWS credentials investigation**:
   - Verify if `aws.accesskey`/`aws.secretaccesskey` are actually used for OSD operations
   - If unused, remove these fields and update `AWSConfig` struct
   - If used, ensure proper credentials are provided

2. **Central IDP configuration review**:
   - Verify if static IDP configuration is still required
   - Consider environment-specific defaults instead of secrets

### Phase 3: Hardcode Static Values (Medium-term)
1. **Red Hat SSO client ID**:
   - Move `redhatsso-service.clientId` from secrets to hardcoded value in `pkg/client/iam/config.go`
   - Update configuration loading to use hardcoded value

2. **Review other static configurations**:
   - Identify additional values that are identical across environments
   - Evaluate for hardcoding opportunities

## Impact Assessment

### Benefits
- **Reduced Secret Management Overhead**: Fewer secrets to manage across environments
- **Improved Security**: Elimination of dummy/placeholder values that could mask real security issues
- **Simplified Configuration**: Cleaner vault configurations with only necessary secrets
- **Reduced Confusion**: Elimination of unused/dummy values that confuse developers

### Risks
- **Service Disruption**: Removing secrets that are used by undocumented code paths
- **Environment Drift**: Changes must be coordinated across all environments

### Mitigation Strategies
- **Comprehensive Testing**: Test changes in integration environment first
- **Gradual Rollout**: Implement changes in phases with rollback plans
- **Documentation Updates**: Update configuration documentation to reflect changes

## Success Metrics

1. **Configuration Complexity Reduction**:
   - Target: Remove 6-8 unused secret keys (40-50% reduction in unused secrets)
   - Measure: Count of secret keys before/after cleanup

2. **Dummy Value Elimination**:
   - Target: Eliminate all "dummy*" placeholder values
   - Measure: Grep analysis for dummy values in vault configs

3. **Developer Experience**:
   - Target: Cleaner, more understandable secret configurations
   - Measure: Documentation clarity and developer feedback

## Conclusion

This configuration cleanup will significantly reduce the complexity and maintenance overhead of the Fleet Manager secret management while improving security posture by eliminating dummy values and unused configurations. The phased approach ensures safe implementation with minimal risk of service disruption.

The cleanup aligns with the existing configuration optimization opportunities identified in my Phase 3 analysis, particularly addressing unused configuration fields and improving secret management security.
