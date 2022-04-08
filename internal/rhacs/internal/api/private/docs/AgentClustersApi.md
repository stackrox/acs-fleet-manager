# \AgentClustersApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetDataPlaneClusterAgentConfig**](AgentClustersApi.md#GetDataPlaneClusterAgentConfig) | **Get** /api/dinosaurs_mgmt/v1/agent-clusters/{id} | Get the data plane cluster agent configuration
[**GetDinosaurs**](AgentClustersApi.md#GetDinosaurs) | **Get** /api/dinosaurs_mgmt/v1/agent-clusters/{id}/dinosaurs | Get the list of ManagedaDinosaurs for the specified agent cluster
[**UpdateAgentClusterStatus**](AgentClustersApi.md#UpdateAgentClusterStatus) | **Put** /api/dinosaurs_mgmt/v1/agent-clusters/{id}/status | Update the status of an agent cluster
[**UpdateDinosaurClusterStatus**](AgentClustersApi.md#UpdateDinosaurClusterStatus) | **Put** /api/dinosaurs_mgmt/v1/agent-clusters/{id}/dinosaurs/status | Update the status of Dinosaur clusters on an agent cluster



## GetDataPlaneClusterAgentConfig

> DataplaneClusterAgentConfig GetDataPlaneClusterAgentConfig(ctx, id)

Get the data plane cluster agent configuration

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 

### Return type

[**DataplaneClusterAgentConfig**](DataplaneClusterAgentConfig.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetDinosaurs

> ManagedDinosaurList GetDinosaurs(ctx, id)

Get the list of ManagedaDinosaurs for the specified agent cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 

### Return type

[**ManagedDinosaurList**](ManagedDinosaurList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateAgentClusterStatus

> UpdateAgentClusterStatus(ctx, id, dataPlaneClusterUpdateStatusRequest)

Update the status of an agent cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
**dataPlaneClusterUpdateStatusRequest** | [**DataPlaneClusterUpdateStatusRequest**](DataPlaneClusterUpdateStatusRequest.md)| Cluster status update data | 

### Return type

 (empty response body)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateDinosaurClusterStatus

> UpdateDinosaurClusterStatus(ctx, id, requestBody)

Update the status of Dinosaur clusters on an agent cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
**requestBody** | [**map[string]DataPlaneDinosaurStatus**](DataPlaneDinosaurStatus.md)| Dinosaur clusters status update data | 

### Return type

 (empty response body)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

