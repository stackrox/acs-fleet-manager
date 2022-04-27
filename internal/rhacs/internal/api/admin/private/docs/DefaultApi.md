# \DefaultApi

All URIs are relative to *https://api.openshift.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteDinosaurById**](DefaultApi.md#DeleteDinosaurById) | **Delete** /api/dinosaurs_mgmt/v1/admin/dinosaurs/{id} | Delete a Dinosaur by ID
[**GetDinosaurById**](DefaultApi.md#GetDinosaurById) | **Get** /api/dinosaurs_mgmt/v1/admin/dinosaurs/{id} | Return the details of Dinosaur instance by id
[**GetDinosaurs**](DefaultApi.md#GetDinosaurs) | **Get** /api/dinosaurs_mgmt/v1/admin/dinosaurs | Returns a list of Dinosaurs
[**UpdateDinosaurById**](DefaultApi.md#UpdateDinosaurById) | **Patch** /api/dinosaurs_mgmt/v1/admin/dinosaurs/{id} | Update a Dinosaur instance by id



## DeleteDinosaurById

> Dinosaur DeleteDinosaurById(ctx, id, async)

Delete a Dinosaur by ID

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
**async** | **bool**| Perform the action in an asynchronous manner | 

### Return type

[**Dinosaur**](Dinosaur.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetDinosaurById

> Dinosaur GetDinosaurById(ctx, id)

Return the details of Dinosaur instance by id

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 

### Return type

[**Dinosaur**](Dinosaur.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetDinosaurs

> DinosaurList GetDinosaurs(ctx, optional)

Returns a list of Dinosaurs

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***GetDinosaursOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a GetDinosaursOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **optional.String**| Name of the request | 
 **ownerUser** | **optional.String**| User that owns the request | 
 **page** | **optional.String**| Page index | 
 **size** | **optional.String**| Number of items in each page | 

### Return type

[**DinosaurList**](DinosaurList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateDinosaurById

> Dinosaur UpdateDinosaurById(ctx, id, dinosaurUpdateRequest)

Update a Dinosaur instance by id

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string**| The ID of record | 
**dinosaurUpdateRequest** | [**DinosaurUpdateRequest**](DinosaurUpdateRequest.md)| Dinosaur update data | 

### Return type

[**Dinosaur**](Dinosaur.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

