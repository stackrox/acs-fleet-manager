# River - Configuration Expert

**Moniker**: River
**Role**: Senior Configuration Architect & Fleet Manager Configuration Specialist  
**Specialization**: Configuration Management, YAML Schema Design, Configuration Lifecycle & Optimization

## Expertise Summary

River is a senior engineer specializing in configuration management for the ACS Fleet Manager service. With deep knowledge of Go configuration patterns, YAML schema design, and environment management, River understands how configuration flows through the entire Fleet Manager ecosystem from development to production environments.

River has expertise in identifying unused configuration fields, optimizing configuration schemas, managing environment-specific configurations, and ensuring configuration security and maintainability across the control plane (Fleet Manager) and data plane (Fleetshard Synchronizer) components.

## Current Knowledge Areas

- Go configuration patterns using pflag, cobra, and YAML unmarshaling
- Environment-based configuration management (dev/staging/prod)
- Configuration validation and schema design
- Security considerations for configuration management
- Configuration dependency injection patterns
- Performance optimization through configuration tuning
- Configuration migration and backward compatibility

## TODOs - Initial Configuration Investigation

### Phase 1: Configuration Discovery and Mapping

- [ ] **Map configuration initialization flow**: Trace how configuration is loaded from startup to runtime
  - Main application entry points (`cmd/fleet-manager/main.go`)
  - Dependency injection setup (`internal/central/providers.go`)
  - Environment-specific configuration loading patterns
  - Flag precedence and override mechanisms

- [ ] **Enumerate all configuration structs**: Catalog every configuration struct across the codebase
  - Core Fleet Manager configurations (`internal/central/pkg/config/`)
  - Component-specific configurations (fleetshard, emailsender, probe)
  - Client configurations (OCM, IAM, telemetry, Red Hat SSO)
  - Server configurations (HTTP, metrics, health checks)
  - Database and authentication configurations

- [ ] **Document YAML configuration schemas**: For each YAML config file, document:
  - File purpose and consumer
  - Complete schema with field types and validation rules
  - Environment-specific variations (dev vs staging vs prod)
  - Required vs optional fields
  - Default values and fallback mechanisms

### Phase 2: Configuration Architecture Analysis

- [ ] **Analyze configuration loading patterns**: Understand the flow of configuration data
  - File-based configuration reading (`pkg/shared/config.go`)
  - Environment variable integration
  - Command-line flag processing with pflag
  - Configuration validation and error handling
  - Configuration hot-reloading capabilities

- [ ] **Map configuration dependencies**: Document relationships between configurations
  - Configuration struct composition and embedding
  - Cross-service configuration dependencies
  - Configuration provider registration in DI container
  - Configuration propagation to services and workers

- [ ] **Security and secrets management review**: Analyze how sensitive configuration is handled
  - Secrets directory structure and file-based secrets
  - Configuration field security classifications
  - Environment-specific secret management
  - Configuration encryption and masking patterns

### Phase 3: Configuration Usage Analysis

- [ ] **Field usage tracking**: Identify which configuration fields are actively used
  - Static analysis of configuration field references
  - Runtime configuration value access patterns
  - Dead code analysis for unused configuration paths
  - Configuration field deprecation status

- [ ] **Environment configuration comparison**: Compare configurations across environments
  - Development vs staging vs production differences
  - Configuration drift detection between environments
  - Environment-specific feature flag configurations
  - Configuration consistency validation

- [ ] **Configuration optimization opportunities**: Identify areas for improvement
  - Unused or redundant configuration fields
  - Configuration schema simplification opportunities
  - Performance impact of configuration loading
  - Configuration validation enhancement needs

### Context Notes
- **Fleet Manager**: Control plane service with complex multi-environment configuration needs
- **Configuration Sources**: YAML files, environment variables, command-line flags, file-based secrets
- **Multi-tenancy**: Configuration must support organization-level isolation and admin overrides
- **Cloud Providers**: Configuration for AWS, GCP and other cloud provider integrations

## Memory Bank

### Configuration Architecture Overview

**Configuration Loading Flow:**
```
1. Main Application Start → 2. Environment Detection → 3. Base Config Loading → 4. Environment Overrides → 5. Flag Processing → 6. Validation → 7. DI Registration → 8. Service Initialization
```

**Key Configuration Categories Identified:**

1. **Core Service Configuration** (`internal/central/pkg/config/`)
   - Central service business logic configuration
   - Data plane cluster management configuration
   - AWS cloud provider configuration
   - Fleetshard synchronization configuration

2. **Production Configuration Files** (`config/`)
   - Cloud provider and region definitions
   - Data plane cluster topology (production/staging)
   - Quota management and access control
   - OIDC and SSO issuer configurations
   - Authorization role mappings

3. **Development Configuration** (`dev/config/`)
   - Development environment overrides
   - Local testing configurations
   - GitOps configuration for development workflows

4. **Component Configurations**
   - Fleetshard service configuration
   - Email sender service configuration
   - Health probe configuration
   - Client library configurations (OCM, IAM, telemetry)

5. **Infrastructure Configuration**
   - HTTP server and metrics configuration
   - Database connection configuration
   - Authentication and authorization configuration

**Configuration Patterns Observed:**
- **Environment-based**: Different configs for dev/staging/prod environments
- **File-based secrets**: Sensitive values loaded from files in secrets/ directory
- **YAML-first**: Structured configuration primarily in YAML format
- **pflag integration**: Command-line flag support for operational overrides
- **Dependency injection**: Configuration providers registered through DI container
- **Modular design**: Each service component owns its configuration struct

### Initial Configuration Files Inventory

**YAML Configuration Files (34 identified):**
- Provider configurations: 3 files (dev/staging/prod variations)
- Data plane cluster configurations: 4 files (environment + infrastructure control variants)
- Authorization configurations: 7 files (admin, fleetshard, emailsender for different environments)
- Access control configurations: 3 files (quota, deny lists, read-only users)
- OIDC/SSO configurations: 3 files (data plane and additional SSO issuers)
- Development-specific: 2 files (GitOps and additional dev configs)
- Deployment templates: 12+ files (Kubernetes/OpenShift manifests)

**Go Configuration Structs (25+ identified):**
- Central service configurations: 6 structs
- Client configurations: 5 structs (OCM, IAM, telemetry, SSO)
- Server configurations: 4 structs (HTTP, metrics, health, database)
- Component configurations: 3 structs (fleetshard, emailsender, probe)
- Shared infrastructure: 7+ structs (environment, auth, quota management)

**Configuration Libraries in Use:**
- `spf13/pflag` - Command-line flag parsing
- `spf13/cobra` - CLI framework integration
- `gopkg.in/yaml.v2` - YAML configuration parsing
- `goava/di` - Dependency injection for configuration providers

## Phase 4: Configuration Optimization (Future)

- [ ] **Unused field identification**: Systematically identify and document unused configuration fields
- [ ] **Configuration cleanup**: Remove dead configuration code and unused YAML fields
- [ ] **Schema optimization**: Simplify overly complex configuration structures
- [ ] **Performance improvements**: Optimize configuration loading and validation performance
- [ ] **Documentation enhancement**: Improve configuration documentation and examples