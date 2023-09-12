# Quota Service Implementations

## Why

To allow us easily open source the project. With the right interface/contract in place for quota management,
The AMS (`ams` `QuotaType`) backed quota management implementation is an implementation detail of the interface.
We have also provided another implementation based on the [quota-management-list](../../config/quota-management-list-configuration.yaml).  

## How

When it is enabled, the following diagram describes the architecture for quota management service:

![Quota Service Interface](../images/quota-service.png)

The `QuotaService` is defined in the [services package](../../internal/dinosaur/pkg/services/quota.go).

The `QuotaServiceFactory` provides the concrete implementation of the `QuotaService` to be used.
The decision is based on the type provided - an enum, currently accepting `ams` and `quota-management-list`.
- The `ams` quota service is implemented using OCM. The implementation can be found in [ams_quota_service.go](../../internal/dinosaur/pkg/services/quota/ams_quota_service.go)
- The `quota-management-list` quota service is implemented using the quota list configuration. The implementation can be found in [quota_management_list_service.go](../../internal/dinosaur/pkg/services/quota/quota_management_list_service.go).
   The quota list based quota service can be disabled by setting the flag `enable-instance-limit-control` to `false`.


New implementations of the `QuotaService` interface can then be added to support different mechanisms to manage quota.
