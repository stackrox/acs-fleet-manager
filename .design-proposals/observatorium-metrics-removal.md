# Design Proposal: Remove Deprecated Observatorium Metrics Feature

## Summary

This proposal outlines the removal of the deprecated Observatorium metrics functionality from the ACS Fleet Manager. The feature includes OpenAPI schema definitions for exposing Prometheus metrics via Fleet Manager API endpoints, but the actual implementation (handlers, routes, services) was never completed. This cleanup will remove unused code and reduce maintenance overhead.

## Background

The ACS Fleet Manager OpenAPI specifications contain comprehensive schema definitions for metrics endpoints that would theoretically interact with Observatorium (Red Hat's managed Prometheus service). However, after thorough analysis, these schemas are not connected to any actual API endpoints, handlers, or service implementations. This indicates the feature was planned but never fully implemented.

## Current State Analysis

### OpenAPI Schema Definitions (Exist but Unused)

The following schemas are defined in the OpenAPI specification but have no corresponding API endpoints or handlers:

**File**: `openapi/fleet-manager.yaml`
- **Lines 700-718**: `MetricsRangeQueryList` schema definition
- **Lines 741-758**: `MetricsInstantQueryList` schema definition  
- **Lines 759-760**: `MetricsFederationResult` schema definition
- **Lines 1038-1058**: Example metric data (`MetricsRangeQueryExample`, `MetricsInstantQueryExample`)

### Generated Model Files (Auto-generated, No Usage)

The following Go model files are generated from the OpenAPI schemas but not used anywhere in the codebase:

- `internal/central/pkg/api/public/model_metrics_range_query_list.go` - MetricsRangeQueryList struct
- `internal/central/pkg/api/public/model_metrics_instant_query_list.go` - MetricsInstantQueryList struct
- `internal/central/pkg/api/public/model_range_query.go` - RangeQuery struct
- `internal/central/pkg/api/public/model_instant_query.go` - InstantQuery struct
- `internal/central/pkg/api/public/model_values.go` - Values struct

### Configuration References

**File**: `dev/env/manifests/addons/crds/00-addon-crd.yaml`
- **Lines 385-387**: Contains Observatorium endpoint URLs for staging and production environments
  ```yaml
  - Staging: https://observatorium-mst.stage.api.openshift.com/api/metrics/v1/<tenant id>/api/v1/receive 
  - Production: https://observatorium-mst.api.openshift.com/api/metrics/v1/<tenant id>/api/v1/receive
  ```

### Authentication Configuration

**Files**: 
- `secrets/rhsso-metrics.clientId` - Empty secret file for Observatorium authentication
- `secrets/rhsso-metrics.clientSecret` - Empty secret file for Observatorium authentication
- `Makefile:630-631` - References to the above secret files in the `secrets/touch` target

### Misleading Code Comment

**File**: `pkg/metrics/metrics.go`
- **Line 664**: Contains a misleading comment referencing "observatorium request duration metric" but this is actually for database query metrics, not Observatorium API access

## Missing Implementation (Confirms Feature is Incomplete)

The following components are **NOT present** in the codebase, confirming the feature is incomplete:

1. **No API endpoints** - No routes defined for metrics endpoints in route configuration
2. **No handlers** - No HTTP handlers to process metrics requests  
3. **No services** - No service layer to interact with Observatorium
4. **No client code** - No code to authenticate with or query Observatorium APIs
5. **No middleware** - No authentication or authorization middleware for metrics endpoints
6. **No usage** - The generated model structs are never imported or used

## Removal Plan

### Phase 1: OpenAPI Schema Cleanup

Remove the following sections from `openapi/fleet-manager.yaml`:

1. **Lines 700-760**: Remove all metrics-related schema definitions:
   - `MetricsRangeQueryList`
   - `MetricsInstantQueryList`
   - `MetricsFederationResult`
   - `RangeQuery`
   - `InstantQuery`
   - `values`

2. **Lines 1038-1058**: Remove example metric data:
   - `MetricsRangeQueryExample`
   - `MetricsInstantQueryExample`

### Phase 2: Generated Code Cleanup

The following files will be automatically removed during the next `make generate` run after the OpenAPI schema changes:

- `internal/central/pkg/api/public/model_metrics_range_query_list.go`
- `internal/central/pkg/api/public/model_metrics_instant_query_list.go`
- `internal/central/pkg/api/public/model_range_query.go`
- `internal/central/pkg/api/public/model_instant_query.go`
- `internal/central/pkg/api/public/model_values.go`

### Phase 3: Configuration Cleanup

1. **Remove authentication secrets**:
   - Delete `secrets/rhsso-metrics.clientId`
   - Delete `secrets/rhsso-metrics.clientSecret`
   - Update `Makefile` lines 630-631 to remove references to these files

2. **Update CRD files** (Optional):
   - Consider removing Observatorium endpoint references from `dev/env/manifests/addons/crds/00-addon-crd.yaml` lines 385-387 if they're not used by other components

### Phase 4: Code Comment Cleanup

1. **Fix misleading comment in `pkg/metrics/metrics.go`**:
   - **Line 664**: Update comment from "Update the observatorium request duration metric" to "Update the database query duration metric"

## Impact Assessment

### Risk Level: **LOW**

- **No functional impact**: The feature was never implemented, so removal cannot break existing functionality
- **No API compatibility issues**: No actual API endpoints exist to be removed
- **No user impact**: No users are consuming metrics through Fleet Manager API endpoints
- **No data loss**: No metrics data is stored or managed by Fleet Manager

### Benefits

1. **Reduced maintenance overhead**: Fewer files to maintain and test
2. **Cleaner codebase**: Removal of unused/incomplete features
3. **Clearer intent**: Eliminates confusion about unimplemented features
4. **Smaller binary size**: Removal of unused generated code

### Dependencies

- This removal requires no coordination with other teams since the feature is not implemented
- No migration of existing users needed
- The internal Prometheus metrics collection (for Fleet Manager's own metrics) remains unaffected

## Testing

### Verification Steps

1. **Build verification**: Ensure `make all` completes successfully after changes
2. **OpenAPI validation**: Run `make openapi/validate` to ensure schema is still valid
3. **Generated code verification**: Run `make generate` and verify no metrics-related models are generated
4. **Secret verification**: Ensure application starts without requiring the removed secret files

### Regression Testing

- Verify all existing API endpoints continue to function
- Verify internal Prometheus metrics (for Fleet Manager monitoring) still work
- Verify no broken imports or references to removed model files

## Future Considerations

If Observatorium metrics functionality is needed in the future:

1. **Implement complete feature**: Include handlers, services, authentication, and routes
2. **Consider alternative approaches**: Direct Observatorium access might be preferred over proxying through Fleet Manager
3. **Security review**: Ensure proper authentication and authorization for metrics access
4. **Performance considerations**: Evaluate impact of proxying metrics queries through Fleet Manager

## Implementation Timeline

- **Immediate**: This cleanup can be implemented immediately with no coordination needed
- **Estimated effort**: 1-2 hours for implementation and testing
- **No feature freeze concerns**: Since feature was never implemented, removal has no compatibility impact

## Conclusion

The removal of the incomplete Observatorium metrics feature will simplify the codebase without any functional impact. The feature appears to have been planned but never fully implemented, leaving only unused schema definitions and generated code. This cleanup aligns with best practices of removing dead code and reducing maintenance overhead.