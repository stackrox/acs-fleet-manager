# Dinosaur

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
**Owner** | **string** |  | [optional] 
**Name** | **string** |  | [optional] 
**Host** | **string** |  | [optional] 
**CreatedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**UpdatedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**FailedReason** | **string** |  | [optional] 
**ActualDinosaurVersion** | **string** |  | [optional] 
**ActualDinosaurOperatorVersion** | **string** |  | [optional] 
**DesiredDinosaurVersion** | **string** |  | [optional] 
**DesiredDinosaurOperatorVersion** | **string** |  | [optional] 
**DinosaurUpgrading** | **bool** |  | 
**DinosaurOperatorUpgrading** | **bool** |  | 
**OrganisationId** | **string** |  | [optional] 
**SubscriptionId** | **string** |  | [optional] 
**OwnerAccountId** | **string** |  | [optional] 
**AccountNumber** | **string** |  | [optional] 
**InstanceType** | **string** |  | [optional] 
**QuotaType** | **string** |  | [optional] 
**Routes** | [**[]DinosaurDetailsRoutes**](DinosaurDetails_routes.md) |  | [optional] 
**RoutesCreated** | **bool** |  | [optional] 
**ClusterId** | **string** |  | [optional] 
**Namespace** | **string** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


