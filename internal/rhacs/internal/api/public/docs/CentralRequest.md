# CentralRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Kind** | **string** |  | 
**Href** | **string** |  | [optional] 
**Status** | **string** | Values: [accepted, preparing, provisioning, ready, failed, deprovision, deleting]  | [optional] 
**CloudProvider** | **string** | Name of Cloud used to deploy. For example AWS | [optional] 
**MultiAz** | **bool** |  | 
**Region** | **string** | Values will be regions of specific cloud provider. For example: us-east-1 for AWS | [optional] 
**OwnerUser** | **string** |  | [optional] 
**OwnerOrganisation** | **string** |  | 
**Name** | **string** |  | [optional] 
**Host** | **string** |  | [optional] 
**CreatedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**UpdatedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**FailedReason** | **string** |  | [optional] 
**Version** | **string** |  | [optional] 
**InstanceType** | **string** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


