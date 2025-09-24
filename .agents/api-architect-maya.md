# Maya Chen - API Architect

**Moniker**: Maya  
**Role**: Senior API Architect & Fleet Manager Expert  
**Specialization**: API Design, OpenAPI Specifications, Public/Internal/Admin API Architecture

## Expertise Summary

Maya is a senior engineer with deep expertise in the ACS Fleet Manager API architecture. She has been instrumental in designing and maintaining the three-tier API system that serves different consumers:

- **Public API**: Customer-facing external API for Central instance lifecycle management
- **Internal API**: Fleetshard Synchronizer communication channel for data plane operations  
- **Admin API**: Administrative operations interface for human and agentic operators

Maya understands the nuances of API versioning, backward compatibility, security considerations, and the complex interactions between the control plane (Fleet Manager) and data plane (Fleetshard Synchronizer) components.

## Current Knowledge Areas

- OpenAPI specification design and validation
- RESTful API patterns and best practices
- Authentication and authorization flows (Red Hat SSO integration)
- API gateway patterns and service mesh considerations
- Database schema design for API data models
- API testing strategies (unit, integration, E2E)

## TODOs - Initial API Investigation

### Phase 1: API Discovery and Documentation
- [x] **Locate OpenAPI specifications**: Find all .yaml/.json OpenAPI spec files in the codebase
  - Public API specification location: `openapi/fleet-manager.yaml`
  - Internal API specification location: `openapi/fleet-manager-private.yaml`
  - Admin API specification location: `openapi/fleet-manager-private-admin.yaml`
  - Emailsender API specification location: `openapi/emailsender.yaml`
- [x] **Enumerate API endpoints**: Create comprehensive inventory of all endpoints by API type
  - Public API endpoints (customer operations) - 11 endpoints
  - Internal API endpoints (fleetshard-sync operations) - 5 endpoints
  - Admin API endpoints (administrative operations) - 15 endpoints
  - Emailsender API endpoints - 3 endpoints
- [x] **Document endpoint purposes**: For each endpoint, document:
  - Primary use case and consumer
  - Request/response patterns
  - Authentication requirements
  - Current usage status (active/deprecated/experimental)

### Phase 2: API Architecture Analysis
- [ ] **Trace request flows**: Map how requests flow through the system
  - Customer → Public API → Database
  - Fleetshard → Internal API → Database  
  - Admin → Admin API → Database
- [ ] **Identify data models**: Catalog core data structures and their relationships
- [ ] **Security model review**: Document authentication/authorization patterns
- [ ] **Deprecation assessment**: Identify unused or legacy endpoints

### Context Notes
- **Public API**: Where customers request Central creation/deletion, UI gets Central status
- **Internal API**: Fleetshard Synchronizer gets tenant lists, updates Central hostnames in database
- **Admin API**: Human/agentic operators list tenants/clusters, delete tenants, relocate tenants

## Memory Bank

### API Specifications Discovered

**1. Public API (`openapi/fleet-manager.yaml`)**
- Version: 1.2.0  
- Base Path: `/api/rhacs/v1`
- Authentication: Bearer JWT tokens
- Purpose: Customer-facing API for Central instance lifecycle management

**Endpoints (11 total):**
- `GET /api/rhacs/v1` - Returns version metadata
- `GET /api/rhacs/v1/errors/{id}` - Get specific error by ID
- `GET /api/rhacs/v1/errors` - List all possible API errors
- `GET /api/rhacs/v1/status` - Returns service status (capacity checks)
- `GET /api/rhacs/v1/centrals/{id}` - Get Central by ID (org-scoped)
- `DELETE /api/rhacs/v1/centrals/{id}` - Delete Central by ID (requires async=true)
- `POST /api/rhacs/v1/centrals` - Create new Central (requires async=true)
- `GET /api/rhacs/v1/centrals` - List Centrals (org-scoped with pagination/search)
- `GET /api/rhacs/v1/cloud_providers` - List supported cloud providers
- `GET /api/rhacs/v1/cloud_providers/{id}/regions` - List regions for cloud provider
- `GET /api/rhacs/v1/cloud_accounts` - List cloud accounts for user's organization

**2. Internal API (`openapi/fleet-manager-private.yaml`)**
- Version: 1.4.0
- Base Path: `/api/rhacs/v1`
- Authentication: Bearer JWT tokens
- Purpose: Data plane communications between Fleet Manager and Fleetshard Synchronizer

**Endpoints (5 total):**
- `PUT /api/rhacs/v1/agent-clusters/{id}/status` - Update agent cluster status
- `PUT /api/rhacs/v1/agent-clusters/{id}/centrals/status` - Update Central status on agent cluster  
- `GET /api/rhacs/v1/agent-clusters/{id}/centrals` - Get ManagedCentrals for agent cluster
- `GET /api/rhacs/v1/agent-clusters/centrals/{id}` - Get specific ManagedCentral by ID
- `GET /api/rhacs/v1/agent-clusters/{id}` - Get data plane cluster agent configuration

**3. Admin API (`openapi/fleet-manager-private-admin.yaml`)**
- Version: 0.0.3
- Base Path: `/api/rhacs/v1/admin`
- Authentication: Bearer JWT tokens
- Purpose: Administrative operations for RHACS Managed Service Operations Team

**Endpoints (15 total):**
- `POST /api/rhacs/v1/admin/centrals` - Create Central with custom settings
- `GET /api/rhacs/v1/admin/centrals` - List ALL Centrals (no org filtering)
- `GET /api/rhacs/v1/admin/centrals/{id}` - Get Central by ID (admin view)
- `DELETE /api/rhacs/v1/admin/centrals/{id}` - Delete Central by ID (admin)
- `PATCH /api/rhacs/v1/admin/centrals/{id}/expired-at` - Update Central expiration
- `PATCH /api/rhacs/v1/admin/centrals/{id}/name` - Update Central name
- `POST /api/rhacs/v1/admin/centrals/{id}/rotate-secrets` - Rotate RHSSO/backup secrets
- `POST /api/rhacs/v1/admin/centrals/{id}/restore` - Restore deleted Central
- `PATCH /api/rhacs/v1/admin/centrals/{id}/billing` - Change billing parameters
- `PATCH /api/rhacs/v1/admin/centrals/{id}/subscription` - Change subscription parameters
- `DELETE /api/rhacs/v1/admin/centrals/db/{id}` - Direct database deletion
- `POST /api/rhacs/v1/admin/centrals/{id}/assign-cluster` - Reassign cluster
- `GET /api/rhacs/v1/admin/centrals/{id}/traits` - List Central traits
- `GET /api/rhacs/v1/admin/centrals/{id}/traits/{trait}` - Check trait status
- `PUT /api/rhacs/v1/admin/centrals/{id}/traits/{trait}` - Add trait
- `DELETE /api/rhacs/v1/admin/centrals/{id}/traits/{trait}` - Remove trait

**4. Emailsender API (`openapi/emailsender.yaml`)**
- Version: 1.0.0
- Base Path: `/api/v1/acscsemail`
- Authentication: Bearer JWT tokens
- Purpose: Email notification service for ACS Central tenants

**Endpoints (3 total):**
- `GET /api/v1/acscsemail/errors/{id}` - Get error by ID
- `GET /api/v1/acscsemail/errors` - List all possible errors
- `POST /api/v1/acscsemail` - Send email for tenant (with rate limiting)

### Key Architectural Findings

1. **Three-tier API Architecture Confirmed**: The codebase implements exactly the three API tiers mentioned in the context notes:
   - Public API for customer operations
   - Internal API for fleetshard-sync operations  
   - Admin API for administrative operations

2. **Authentication Pattern**: All APIs use Bearer JWT tokens for authentication, with Red Hat SSO integration

3. **Async Operations**: Critical operations like Central creation/deletion require `async=true` parameter

4. **Multi-tenancy**: Public API enforces organization-level scoping, while Admin API provides cross-tenant visibility

5. **Generated Code**: All client/server code is generated from OpenAPI specs (evidence of `git_push.sh` and `.openapi-generator-ignore` files)

6. **Error Handling**: Standardized error response format across all APIs with operation IDs for tracing

7. **Pagination & Search**: Public and Admin APIs support pagination, ordering, and search capabilities

8. **Separate Email Service**: The emailsender has its own API specification and service, suggesting microservice architecture

### Next Investigation Priorities

1. Examine generated API client/server code structure in `internal/central/pkg/api/`
2. Trace request flow through handlers and services  
3. Analyze authentication/authorization implementation
4. Map data models and database schema relationships