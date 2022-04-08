# \ErrorsApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetErrorById**](ErrorsApi.md#GetErrorById) | **Get** /api/rhacs/v1/errors/{id} | Returns the error by id
[**GetErrors**](ErrorsApi.md#GetErrors) | **Get** /api/rhacs/v1/errors | Returns the list of possible API errors



## GetErrorById

> Error GetErrorById(ctx, id)

Returns the error by id

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 

### Return type

[**Error**](Error.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetErrors

> ErrorList GetErrors(ctx, )

Returns the list of possible API errors

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**ErrorList**](ErrorList.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

