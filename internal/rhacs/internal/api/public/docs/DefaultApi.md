# \DefaultApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateCentral**](DefaultApi.md#CreateCentral) | **Post** /api/rhacs/v1/centrals | Creates a Central request
[**DeleteCentralById**](DefaultApi.md#DeleteCentralById) | **Delete** /api/rhacs/v1/centrals/{id} | Deletes a Central request by ID
[**GetCentralById**](DefaultApi.md#GetCentralById) | **Get** /api/rhacs/v1/centrals/{id} | Returns a Central request by ID
[**GetCentrals**](DefaultApi.md#GetCentrals) | **Get** /api/rhacs/v1/centrals | Returns a list of Central requests
[**GetCloudProviderRegions**](DefaultApi.md#GetCloudProviderRegions) | **Get** /api/rhacs/v1/cloud_providers/{id}/regions | Returns the list of supported regions of the supported cloud provider
[**GetCloudProviders**](DefaultApi.md#GetCloudProviders) | **Get** /api/rhacs/v1/cloud_providers | Returns the list of supported cloud providers
[**GetServiceStatus**](DefaultApi.md#GetServiceStatus) | **Get** /api/rhacs/v1/status | Returns the status of resources, such as whether maximum service capacity has been reached
[**GetVersionMetadata**](DefaultApi.md#GetVersionMetadata) | **Get** /api/rhacs/v1 | Returns the version metadata
[**UpdateCentralById**](DefaultApi.md#UpdateCentralById) | **Patch** /api/rhacs/v1/centrals/{id} | Update a Central instance by id



## CreateCentral

> CentralRequest CreateCentral(ctx, async, centralRequestPayload)

Creates a Central request

Each central has a single owner organisation and a single owner user. Creates a new Central that is owned by the user and organisation authenticated for the request.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**async** | **bool**| Perform the action in an asynchronous manner | 
**centralRequestPayload** | [**CentralRequestPayload**](CentralRequestPayload.md)| Central data | 

### Return type

[**CentralRequest**](CentralRequest.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteCentralById

> DeleteCentralById(ctx, id, async)

Deletes a Central request by ID

The only users authorized for this operation are: 1) The administrator of the owner organisation of the specified Central. 2) The owner user, and only if it is also part of the owner organisation of the specified Central. 

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
**async** | **bool**| Perform the action in an asynchronous manner | 

### Return type

 (empty response body)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetCentralById

> CentralRequest GetCentralById(ctx, id)

Returns a Central request by ID

This operation is only authorized to users in the same organisation as the owner organisation of the specified Central.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 

### Return type

[**CentralRequest**](CentralRequest.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetCentrals

> CentralRequestList GetCentrals(ctx, optional)

Returns a list of Central requests

Only returns those centrals that are owned by the organisation of the user authenticated for the request.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***GetCentralsOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a GetCentralsOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **optional.String**| Name of the request | 
 **ownerUser** | **optional.String**| User that owns the request | 
 **pageCursor** | **optional.String**| Page cursor, provided with each page in case more pages are remaining. If missing then the first page is returned. | 
 **size** | **optional.String**| Number of items in each page | 

### Return type

[**CentralRequestList**](CentralRequestList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetCloudProviderRegions

> CloudRegionList GetCloudProviderRegions(ctx, id, optional)

Returns the list of supported regions of the supported cloud provider

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
 **optional** | ***GetCloudProviderRegionsOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a GetCloudProviderRegionsOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **pageCursor** | **optional.String**| Page cursor, provided with each page in case more pages are remaining. If missing then the first page is returned. | 
 **size** | **optional.String**| Number of items in each page | 
 **instanceType** | **optional.String**| The Central instance type to filter the results by | 

### Return type

[**CloudRegionList**](CloudRegionList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetCloudProviders

> CloudProviderList GetCloudProviders(ctx, optional)

Returns the list of supported cloud providers

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***GetCloudProvidersOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a GetCloudProvidersOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pageCursor** | **optional.String**| Page cursor, provided with each page in case more pages are remaining. If missing then the first page is returned. | 
 **size** | **optional.String**| Number of items in each page | 

### Return type

[**CloudProviderList**](CloudProviderList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetServiceStatus

> ServiceStatus GetServiceStatus(ctx, )

Returns the status of resources, such as whether maximum service capacity has been reached

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**ServiceStatus**](ServiceStatus.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetVersionMetadata

> VersionMetadata GetVersionMetadata(ctx, )

Returns the version metadata

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**VersionMetadata**](VersionMetadata.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateCentralById

> CentralRequest UpdateCentralById(ctx, id, centralUpdateRequest)

Update a Central instance by id

The only users authorized for this operation are: 1) The administrator of the owner organisation of the specified Central. 2) The owner user, and only if it is also part of the owner organisation of the specified Central. 

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
**centralUpdateRequest** | [**CentralUpdateRequest**](CentralUpdateRequest.md)| Update owner of Central | 

### Return type

[**CentralRequest**](CentralRequest.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

