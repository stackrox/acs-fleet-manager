// Package errors ...
package errors

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// ErrorCodePrefix ...
const (
	ErrorCodePrefix = "RHACS-MGMT"

	// HREF for API errors
	ErrorHREF = "/api/rhacs/v1/errors/"

	// To support connector errors too..
	ConnectorMgmtErrorCodePrefix = "CONNECTOR-MGMT"
	ConnectorMgmtErrorHREF       = "/api/connector_mgmt/v1/errors/"

	// Forbidden occurs when a user is not allowed to access the service
	ErrorForbidden       ServiceErrorCode = 4
	ErrorForbiddenReason string           = "Forbidden to perform this action"

	// Forbidden occurs when a user or organisation has reached maximum number of allowed instances
	ErrorMaxAllowedInstanceReached       ServiceErrorCode = 5
	ErrorMaxAllowedInstanceReachedReason string           = "Forbidden to create more instances than the maximum allowed"

	// Conflict occurs when a database constraint is violated
	ErrorConflict       ServiceErrorCode = 6
	ErrorConflictReason string           = "An entity with the specified unique values already exists"

	// NotFound occurs when a record is not found in the database
	ErrorNotFound       ServiceErrorCode = 7
	ErrorNotFoundReason string           = "Resource not found"

	// Validation occurs when an object fails validation
	ErrorValidation       ServiceErrorCode = 8
	ErrorValidationReason string           = "General validation failure"

	// General occurs when an error fails to match any other error code
	ErrorGeneral       ServiceErrorCode = 9
	ErrorGeneralReason string           = "Unspecified error"

	// NotImplemented occurs when an API REST method is not implemented in a handler
	ErrorNotImplemented       ServiceErrorCode = 10
	ErrorNotImplementedReason string           = "HTTP Method not implemented for this endpoint"

	// Unauthorized occurs when the requester is not authorized to perform the specified action
	ErrorUnauthorized       ServiceErrorCode = 11
	ErrorUnauthorizedReason string           = "Account is unauthorized to perform this action"

	// Unauthorized occurs when the requester is not authorized to perform the specified action
	ErrorTermsNotAccepted       ServiceErrorCode = 12
	ErrorTermsNotAcceptedReason string           = "Required terms have not been accepted"

	// Unauthenticated occurs when the provided credentials cannot be validated
	ErrorUnauthenticated       ServiceErrorCode = 15
	ErrorUnauthenticatedReason string           = "Account authentication could not be verified"

	// MalformedRequest occurs when the request body cannot be read
	ErrorMalformedRequest       ServiceErrorCode = 17
	ErrorMalformedRequestReason string           = "Unable to read request body"

	// Bad Request
	ErrorBadRequest       ServiceErrorCode = 21
	ErrorBadRequestReason string           = "Bad request"

	// Invalid Search Query
	ErrorFailedToParseSearch       ServiceErrorCode = 23
	ErrorFailedToParseSearchReason string           = "Failed to parse search query"

	// TooManyRequests occurs when a the dinosaur instances capacity gets filled up
	ErrorTooManyDinosaurInstancesReached       ServiceErrorCode = 24
	ErrorTooManyDinosaurInstancesReachedReason string           = "The maximum number of allowed central instances has been reached"

	// Gone occurs when a record is accessed that has been deleted
	ErrorGone       ServiceErrorCode = 25
	ErrorGoneReason string           = "Resource gone"

	// Synchronous request not supported
	ErrorSyncActionNotSupported       ServiceErrorCode = 103
	ErrorSyncActionNotSupportedReason string           = "Synchronous action is not supported, use async=true parameter"

	// Failed to create sso client - an internal error incurred when calling iam server
	ErrorFailedToCreateSSOClient       ServiceErrorCode = 106
	ErrorFailedToCreateSSOClientReason string           = "Failed to create central client in the mas sso"

	// Failed to get sso client secret  - an internal error incurred when calling iam server
	ErrorFailedToGetSSOClientSecret       ServiceErrorCode = 107
	ErrorFailedToGetSSOClientSecretReason string           = "Failed to get central client secret from the mas sso"

	// Failed to get sso client - an internal error incurred when calling iam server
	ErrorFailedToGetSSOClient       ServiceErrorCode = 108
	ErrorFailedToGetSSOClientReason string           = "Failed to get central client from the mas sso"

	// Failed to delete sso client - an internal error incurred when calling iam server
	ErrorFailedToDeleteSSOClient       ServiceErrorCode = 109
	ErrorFailedToDeleteSSOClientReason string           = "Failed to delete central client from the mas sso"

	// Failed to create service account, after validating user's request, but failed at the server end
	// it is an internal server error
	ErrorFailedToCreateServiceAccount       ServiceErrorCode = 110
	ErrorFailedToCreateServiceAccountReason string           = "Failed to create service account"

	// Failed to get service account - an internal error incurred when calling iam server
	ErrorFailedToGetServiceAccount       ServiceErrorCode = 111
	ErrorFailedToGetServiceAccountReason string           = "Failed to get service account"

	// Failed to delete service account - an internal error incurred when calling iam server
	ErrorFailedToDeleteServiceAccount       ServiceErrorCode = 112
	ErrorFailedToDeleteServiceAccountReason string           = "Failed to delete service account"

	// Failed to find service account - a client error as incorrect SA is given
	ErrorServiceAccountNotFound       ServiceErrorCode = 113
	ErrorServiceAccountNotFoundReason string           = "Failed to find service account"

	ErrorMaxLimitForServiceAccountsReached       ServiceErrorCode = 115
	ErrorMaxLimitForServiceAccountsReachedReason string           = "Max limit for the service account creation has reached"

	// Insufficient quota
	ErrorInsufficientQuota       ServiceErrorCode = 120
	ErrorInsufficientQuotaReason string           = "Insufficient quota"
	// Failed to check Quota
	ErrorFailedToCheckQuota       ServiceErrorCode = 121
	ErrorFailedToCheckQuotaReason string           = "Failed to check quota"
	// Provider not supported
	ErrorProviderNotSupported       ServiceErrorCode = 30
	ErrorProviderNotSupportedReason string           = "Provider not supported"
	// Cloud account ID is not set up properly
	ErrorInvalidCloudAccountID       ServiceErrorCode = 122
	ErrorInvalidCloudAccountIDReason string           = "Cloud account ID is not set up properly"

	// Region not supported
	ErrorRegionNotSupported       ServiceErrorCode = 31
	ErrorRegionNotSupportedReason string           = "Region not supported"

	// Invalid dinosaur cluster name
	ErrorMalformedDinosaurClusterName       ServiceErrorCode = 32
	ErrorMalformedDinosaurClusterNameReason string           = "Central cluster name is invalid"

	// Minimum field length validation
	ErrorMinimumFieldLength       ServiceErrorCode = 33
	ErrorMinimumFieldLengthReason string           = "Minimum field length not reached"

	// Maximum field length validation
	ErrorMaximumFieldLength       ServiceErrorCode = 34
	ErrorMaximumFieldLengthReason string           = "Maximum field length has been depassed"

	// Only MultiAZ is supported
	ErrorOnlyMultiAZSupported       ServiceErrorCode = 35
	ErrorOnlyMultiAZSupportedReason string           = "Only multiAZ Centrals are supported, use multi_az=true"

	// Dinosaur cluster name must be unique
	ErrorDuplicateDinosaurClusterName       ServiceErrorCode = 36
	ErrorDuplicateDinosaurClusterNameReason string           = "Central cluster name is already used"

	// A generic field validation error when validating API requests input
	ErrorFieldValidationError       ServiceErrorCode = 37
	ErrorFieldValidationErrorReason string           = "Field validation failed"

	// Failure to send an error response (i.e. unable to send error response as the error can't be converted to JSON.)
	ErrorUnableToSendErrorResponse       ServiceErrorCode = 1000
	ErrorUnableToSendErrorResponseReason string           = "An unexpected error happened, please check the log of the service for details"

	// Invalid service account name
	ErrorMalformedServiceAccountName       ServiceErrorCode = 38
	ErrorMalformedServiceAccountNameReason string           = "Service account name is invalid"

	// Invalid service account desc
	ErrorMalformedServiceAccountDesc       ServiceErrorCode = 39
	ErrorMalformedServiceAccountDescReason string           = "Service account desc is invalid"

	// Invalid service account desc
	ErrorMalformedServiceAccountID       ServiceErrorCode = 40
	ErrorMalformedServiceAccountIDReason string           = "Service account id is invalid"

	// Region not supported
	ErrorInstanceTypeNotSupported       ServiceErrorCode = 41
	ErrorInstanceTypeNotSupportedReason string           = "Instance Type not supported"

	// Instance plan not supported
	ErrorInstancePlanNotSupported       ServiceErrorCode = 42
	ErrorInstancePlanNotSupportedReason string           = "Instance plan not supported"

	// Too Many requests error. Used by rate limiting
	ErrorTooManyRequests       ServiceErrorCode = 429
	ErrorTooManyRequestsReason string           = "Too Many requests"
)

// ErrorList ...
type ErrorList []error

// Error ...
func (e ErrorList) Error() string {
	var res string
	for _, err := range e {
		res = res + fmt.Sprintf(";%s", err)
	}

	res = fmt.Sprintf("[%s]", res)
	return res
}

// ServiceErrorCode ...
type ServiceErrorCode int

// ServiceErrors ...
type ServiceErrors []ServiceError

// Find ...
func Find(code ServiceErrorCode) (bool, *ServiceError) {
	for _, err := range Errors() {
		if err.Code == code {
			return true, &err
		}
	}
	return false, nil
}

// Errors ...
func Errors() ServiceErrors {
	return ServiceErrors{
		ServiceError{ErrorForbidden, ErrorForbiddenReason, http.StatusForbidden, nil},
		ServiceError{ErrorMaxAllowedInstanceReached, ErrorMaxAllowedInstanceReachedReason, http.StatusForbidden, nil},
		ServiceError{ErrorTooManyDinosaurInstancesReached, ErrorTooManyDinosaurInstancesReachedReason, http.StatusForbidden, nil},
		ServiceError{ErrorTooManyRequests, ErrorTooManyRequestsReason, http.StatusTooManyRequests, nil},
		ServiceError{ErrorConflict, ErrorConflictReason, http.StatusConflict, nil},
		ServiceError{ErrorNotFound, ErrorNotFoundReason, http.StatusNotFound, nil},
		ServiceError{ErrorGone, ErrorGoneReason, http.StatusGone, nil},
		ServiceError{ErrorValidation, ErrorValidationReason, http.StatusBadRequest, nil},
		ServiceError{ErrorGeneral, ErrorGeneralReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorNotImplemented, ErrorNotImplementedReason, http.StatusMethodNotAllowed, nil},
		ServiceError{ErrorUnauthorized, ErrorUnauthorizedReason, http.StatusForbidden, nil},
		ServiceError{ErrorTermsNotAccepted, ErrorTermsNotAcceptedReason, http.StatusForbidden, nil},
		ServiceError{ErrorUnauthenticated, ErrorUnauthenticatedReason, http.StatusUnauthorized, nil},
		ServiceError{ErrorMalformedRequest, ErrorMalformedRequestReason, http.StatusBadRequest, nil},
		ServiceError{ErrorBadRequest, ErrorBadRequestReason, http.StatusBadRequest, nil},
		ServiceError{ErrorFailedToParseSearch, ErrorFailedToParseSearchReason, http.StatusBadRequest, nil},
		ServiceError{ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason, http.StatusBadRequest, nil},
		ServiceError{ErrorFailedToCreateSSOClient, ErrorFailedToCreateSSOClientReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorFailedToGetSSOClientSecret, ErrorFailedToGetSSOClientSecretReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorFailedToGetSSOClient, ErrorFailedToGetSSOClientReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorFailedToDeleteSSOClient, ErrorFailedToDeleteSSOClientReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorFailedToCreateServiceAccount, ErrorFailedToCreateServiceAccountReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorFailedToGetServiceAccount, ErrorFailedToGetServiceAccountReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorServiceAccountNotFound, ErrorServiceAccountNotFoundReason, http.StatusNotFound, nil},
		ServiceError{ErrorFailedToDeleteServiceAccount, ErrorFailedToDeleteServiceAccountReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorProviderNotSupported, ErrorProviderNotSupportedReason, http.StatusBadRequest, nil},
		ServiceError{ErrorRegionNotSupported, ErrorRegionNotSupportedReason, http.StatusBadRequest, nil},
		ServiceError{ErrorInstanceTypeNotSupported, ErrorInstanceTypeNotSupportedReason, http.StatusBadRequest, nil},
		ServiceError{ErrorMalformedDinosaurClusterName, ErrorMalformedDinosaurClusterNameReason, http.StatusBadRequest, nil},
		ServiceError{ErrorMinimumFieldLength, ErrorMinimumFieldLengthReason, http.StatusBadRequest, nil},
		ServiceError{ErrorMaximumFieldLength, ErrorMaximumFieldLengthReason, http.StatusBadRequest, nil},
		ServiceError{ErrorOnlyMultiAZSupported, ErrorOnlyMultiAZSupportedReason, http.StatusBadRequest, nil},
		ServiceError{ErrorDuplicateDinosaurClusterName, ErrorDuplicateDinosaurClusterNameReason, http.StatusConflict, nil},
		ServiceError{ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorFieldValidationError, ErrorFieldValidationErrorReason, http.StatusBadRequest, nil},
		ServiceError{ErrorInsufficientQuota, ErrorInsufficientQuotaReason, http.StatusForbidden, nil},
		ServiceError{ErrorFailedToCheckQuota, ErrorFailedToCheckQuotaReason, http.StatusInternalServerError, nil},
		ServiceError{ErrorMalformedServiceAccountName, ErrorMalformedServiceAccountNameReason, http.StatusBadRequest, nil},
		ServiceError{ErrorMalformedServiceAccountDesc, ErrorMalformedServiceAccountDescReason, http.StatusBadRequest, nil},
		ServiceError{ErrorMalformedServiceAccountID, ErrorMalformedServiceAccountIDReason, http.StatusBadRequest, nil},
		ServiceError{ErrorMaxLimitForServiceAccountsReached, ErrorMaxLimitForServiceAccountsReachedReason, http.StatusForbidden, nil},
		ServiceError{ErrorInstancePlanNotSupported, ErrorInstancePlanNotSupportedReason, http.StatusBadRequest, nil},
		ServiceError{ErrorInvalidCloudAccountID, ErrorInvalidCloudAccountIDReason, http.StatusBadRequest, nil},
	}
}

// NewErrorFromHTTPStatusCode ...
func NewErrorFromHTTPStatusCode(httpCode int, reason string, values ...interface{}) *ServiceError {
	if httpCode >= http.StatusBadRequest && httpCode < http.StatusInternalServerError {
		switch httpCode {
		case http.StatusBadRequest:
			return BadRequest(reason, values...)
		case http.StatusUnauthorized:
			return Unauthorized(reason, values...)
		case http.StatusForbidden:
			return Forbidden(reason, values...)
		case http.StatusNotFound:
			return NotFound(reason, values...)
		case http.StatusMethodNotAllowed:
			return NotImplemented(reason, values...)
		case http.StatusConflict:
			return Conflict(reason, values...)
		default:
			return BadRequest(reason, values...)
		}
	}

	if httpCode >= http.StatusInternalServerError {
		switch httpCode {
		case http.StatusInternalServerError:
			return GeneralError(reason, values...)
		default:
			return GeneralError(reason, values...)
		}
	}

	return GeneralError(reason, values...)
}

// ToServiceError ...
func ToServiceError(err error) *ServiceError {
	switch convertedErr := err.(type) {
	case *ServiceError:
		return convertedErr
	default:
		return GeneralError(convertedErr.Error())
	}
}

// ServiceError ...
type ServiceError struct {
	// Code is the numeric and distinct ID for the error
	Code ServiceErrorCode
	// Reason is the context-specific reason the error was generated
	Reason string
	// HTTPCode is the HTTPCode associated with the error when the error is returned as an API response
	HTTPCode int
	// The original error that is causing the ServiceError, can be used for inspection
	cause error
}

// New Reason can be a string with format verbs, which will be replace by the specified values
func New(code ServiceErrorCode, reason string, values ...interface{}) *ServiceError {
	return NewWithCause(code, nil, reason, values...)
}

// NewWithCause ...
func NewWithCause(code ServiceErrorCode, cause error, reason string, values ...interface{}) *ServiceError {
	// If the code isn't defined, use the general error code
	var err *ServiceError
	exists, err := Find(code)
	if !exists {
		glog.Errorf("Undefined error code used: %d", code)
		err = &ServiceError{ErrorGeneral, "Unspecified error", http.StatusInternalServerError, nil}
	}

	// TODO - if cause is nil, should we use the reason as the cause?
	if cause != nil {
		_, ok := cause.(stackTracer)
		if !ok {
			cause = errors.WithStack(cause) // add stacktrace info
		}
	}
	err.cause = cause

	// If the reason is unspecified, use the default
	if reason != "" {
		err.Reason = fmt.Sprintf(reason, values...)
	}

	return err
}

// Unwrap returns the original error that caused the ServiceError. Can be used with errors.Unwrap.
func (e *ServiceError) Unwrap() error {
	return e.cause
}

// StackTrace returns errors stacktrace.
func (e *ServiceError) StackTrace() errors.StackTrace {
	if e.cause == nil {
		return nil
	}

	err, ok := e.cause.(stackTracer)
	if !ok {
		return nil
	}

	return err.StackTrace()
}

// Error ...
func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %s\n caused by: %s", CodeStr(e.Code), e.Reason, e.cause.Error())
	}

	return fmt.Sprintf("%s: %s", CodeStr(e.Code), e.Reason)
}

// AsError ...
func (e *ServiceError) AsError() error {
	if e == nil {
		return nil
	}
	return fmt.Errorf(e.Error())
}

// Is404 ...
func (e *ServiceError) Is404() bool {
	return e.Code == NotFound("").Code
}

// IsConflict ...
func (e *ServiceError) IsConflict() bool {
	return e.Code == Conflict("").Code
}

// IsForbidden ...
func (e *ServiceError) IsForbidden() bool {
	return e.Code == Forbidden("").Code
}

// IsFailedToCreateSSOClient ...
func (e *ServiceError) IsFailedToCreateSSOClient() bool {
	return e.Code == FailedToCreateSSOClient("").Code
}

// IsClientErrorClass ...
func (e *ServiceError) IsClientErrorClass() bool {
	return e.HTTPCode >= http.StatusBadRequest && e.HTTPCode < http.StatusInternalServerError
}

// IsServerErrorClass ...
func (e *ServiceError) IsServerErrorClass() bool {
	return e.HTTPCode >= http.StatusInternalServerError
}

// IsFailedToGetSSOClientSecret ...
func (e *ServiceError) IsFailedToGetSSOClientSecret() bool {
	return e.Code == FailedToGetSSOClientSecret("").Code
}

// IsFailedToGetSSOClient ...
func (e *ServiceError) IsFailedToGetSSOClient() bool {
	return e.Code == FailedToGetSSOClient("").Code
}

// IsFailedToDeleteSSOClient ...
func (e *ServiceError) IsFailedToDeleteSSOClient() bool {
	return e.Code == FailedToDeleteSSOClient("").Code
}

// IsFailedToCreateServiceAccount ...
func (e *ServiceError) IsFailedToCreateServiceAccount() bool {
	return e.Code == FailedToCreateServiceAccount("").Code
}

// IsFailedToGetServiceAccount ...
func (e *ServiceError) IsFailedToGetServiceAccount() bool {
	return e.Code == FailedToGetServiceAccount("").Code
}

// IsFailedToDeleteServiceAccount ...
func (e *ServiceError) IsFailedToDeleteServiceAccount() bool {
	return e.Code == FailedToDeleteServiceAccount("").Code
}

// IsServiceAccountNotFound ...
func (e *ServiceError) IsServiceAccountNotFound() bool {
	return e.Code == ServiceAccountNotFound("").Code
}

// IsMaxLimitForServiceAccountReached ...
func (e *ServiceError) IsMaxLimitForServiceAccountReached() bool {
	return e.Code == ErrorMaxLimitForServiceAccountsReached
}

// IsBadRequest ...
func (e *ServiceError) IsBadRequest() bool {
	return e.Code == BadRequest("").Code
}

// InSufficientQuota ...
func (e *ServiceError) InSufficientQuota() bool {
	return e.Code == InsufficientQuotaError("").Code
}

// IsFailedToCheckQuota ...
func (e *ServiceError) IsFailedToCheckQuota() bool {
	return e.Code == FailedToCheckQuota("").Code
}

// IsInstanceTypeNotSupported ...
func (e *ServiceError) IsInstanceTypeNotSupported() bool {
	return e.Code == ErrorInstanceTypeNotSupported
}

// AsOpenapiError ...
func (e *ServiceError) AsOpenapiError(operationID string, basePath string) compat.Error {
	href := Href(e.Code)
	code := CodeStr(e.Code)

	if strings.Contains(basePath, "/api/connector_mgmt/") {
		href = strings.Replace(href, ErrorHREF, ConnectorMgmtErrorHREF, 1)
		code = strings.Replace(code, ErrorCodePrefix, ConnectorMgmtErrorCodePrefix, 1)
	}

	// end-temporary code
	return compat.Error{
		Kind:        "Error",
		Id:          strconv.Itoa(int(e.Code)),
		Href:        href,
		Code:        code,
		Reason:      e.Reason,
		OperationId: operationID,
	}
}

// CodeStr ...
func CodeStr(code ServiceErrorCode) string {
	return fmt.Sprintf("%s-%d", ErrorCodePrefix, code)
}

// Href ...
func Href(code ServiceErrorCode) string {
	return fmt.Sprintf("%s%d", ErrorHREF, code)
}

// NotFound ...
func NotFound(reason string, values ...interface{}) *ServiceError {
	return New(ErrorNotFound, reason, values...)
}

// GeneralError ...
func GeneralError(reason string, values ...interface{}) *ServiceError {
	return New(ErrorGeneral, reason, values...)
}

// Unauthorized ...
func Unauthorized(reason string, values ...interface{}) *ServiceError {
	return New(ErrorUnauthorized, reason, values...)
}

// TermsNotAccepted ...
func TermsNotAccepted(reason string, values ...interface{}) *ServiceError {
	return New(ErrorTermsNotAccepted, reason, values...)
}

// Unauthenticated ...
func Unauthenticated(reason string, values ...interface{}) *ServiceError {
	return New(ErrorUnauthenticated, reason, values...)
}

// Forbidden ...
func Forbidden(reason string, values ...interface{}) *ServiceError {
	return New(ErrorForbidden, reason, values...)
}

// MaximumAllowedInstanceReached ...
func MaximumAllowedInstanceReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaxAllowedInstanceReached, reason, values...)
}

// TooManyDinosaurInstancesReached ...
func TooManyDinosaurInstancesReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorTooManyDinosaurInstancesReached, reason, values...)
}

// NotImplemented ...
func NotImplemented(reason string, values ...interface{}) *ServiceError {
	return New(ErrorNotImplemented, reason, values...)
}

// Conflict ...
func Conflict(reason string, values ...interface{}) *ServiceError {
	return New(ErrorConflict, reason, values...)
}

// Validation ...
func Validation(reason string, values ...interface{}) *ServiceError {
	return New(ErrorValidation, reason, values...)
}

// MalformedRequest ...
func MalformedRequest(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedRequest, reason, values...)
}

// BadRequest ...
func BadRequest(reason string, values ...interface{}) *ServiceError {
	return New(ErrorBadRequest, reason, values...)
}

// FailedToParseSearch ...
func FailedToParseSearch(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorFailedToParseSearchReason, reason)
	return New(ErrorFailedToParseSearch, message, values...)
}

// SyncActionNotSupported ...
func SyncActionNotSupported() *ServiceError {
	return New(ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason)
}

// NotMultiAzActionNotSupported ...
func NotMultiAzActionNotSupported() *ServiceError {
	return New(ErrorOnlyMultiAZSupported, ErrorOnlyMultiAZSupportedReason)
}

// FailedToCreateSSOClient ...
func FailedToCreateSSOClient(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToCreateSSOClient, reason, values...)
}

// FailedToGetSSOClientSecret ...
func FailedToGetSSOClientSecret(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetSSOClientSecret, reason, values...)
}

// FailedToGetSSOClient ...
func FailedToGetSSOClient(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetSSOClient, reason, values...)
}

// FailedToDeleteSSOClient ...
func FailedToDeleteSSOClient(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToDeleteSSOClient, reason, values...)
}

// FailedToCreateServiceAccount ...
func FailedToCreateServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToCreateServiceAccount, reason, values...)
}

// FailedToDeleteServiceAccount ...
func FailedToDeleteServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToDeleteServiceAccount, reason, values...)
}

// MaxLimitForServiceAccountReached ...
func MaxLimitForServiceAccountReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaxLimitForServiceAccountsReached, reason, values...)
}

// FailedToGetServiceAccount ...
func FailedToGetServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetServiceAccount, reason, values...)
}

// ServiceAccountNotFound ...
func ServiceAccountNotFound(reason string, values ...interface{}) *ServiceError {
	return New(ErrorServiceAccountNotFound, reason, values...)
}

// RegionNotSupported ...
func RegionNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorRegionNotSupported, reason, values...)
}

// InstanceTypeNotSupported ...
func InstanceTypeNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInstanceTypeNotSupported, reason, values...)
}

// ProviderNotSupported ...
func ProviderNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorProviderNotSupported, reason, values...)
}

// MalformedDinosaurClusterName ...
func MalformedDinosaurClusterName(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedDinosaurClusterName, reason, values...)
}

// InstancePlanNotSupported ...
func InstancePlanNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInstancePlanNotSupported, reason, values...)
}

// MalformedServiceAccountName ...
func MalformedServiceAccountName(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedServiceAccountName, reason, values...)
}

// MalformedServiceAccountDesc ...
func MalformedServiceAccountDesc(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedServiceAccountDesc, reason, values...)
}

// MalformedServiceAccountID ...
func MalformedServiceAccountID(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedServiceAccountID, reason, values...)
}

// DuplicateDinosaurClusterName ...
func DuplicateDinosaurClusterName() *ServiceError {
	return New(ErrorDuplicateDinosaurClusterName, ErrorDuplicateDinosaurClusterNameReason)
}

// MinimumFieldLengthNotReached ...
func MinimumFieldLengthNotReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMinimumFieldLength, reason, values...)
}

// MaximumFieldLengthMissing ...
func MaximumFieldLengthMissing(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaximumFieldLength, reason, values...)
}

// UnableToSendErrorResponse ...
func UnableToSendErrorResponse() *ServiceError {
	return New(ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason)
}

// FailedToParseQueryParms ...
func FailedToParseQueryParms(reason string, values ...interface{}) *ServiceError {
	return New(ErrorBadRequest, reason, values...)

}

// FieldValidationError ...
func FieldValidationError(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorFieldValidationErrorReason, reason)

	return New(ErrorFieldValidationError, message, values...)
}

// InsufficientQuotaError ...
func InsufficientQuotaError(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorInsufficientQuotaReason, reason)
	return New(ErrorInsufficientQuota, message, values...)
}

// FailedToCheckQuota ...
func FailedToCheckQuota(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorFailedToCheckQuotaReason, reason)
	return New(ErrorFailedToCheckQuota, message, values...)
}

// InvalidCloudAccountID ...
func InvalidCloudAccountID(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorInvalidCloudAccountIDReason, reason)
	return New(ErrorInvalidCloudAccountID, message, values...)
}

// InvalidOCMConnection is raised when the OCM connection is invalid.
// Root causes for an invalid connection are invalid credentials or OCM base URLs.
func InvalidOCMConnection() *ServiceError {
	return New(ErrorGeneral, "invalid OCM connection")
}

// OrganisationNotFound converts the error to a ServiceError and returns a reason and hint for the user.
func OrganisationNotFound(externalID string, err error) *ServiceError {
	svcErr := ToServiceError(err)
	reason := "organisation with external id %s not found"
	// Visiting the OpenShift page in console registers the user and their organisation with OCM.
	// See https://issues.redhat.com/browse/SDB-3194 for more context.
	if svcErr.Is404() {
		reason += " - visit https://console.redhat.com/openshift and try again"
	}
	return NewWithCause(svcErr.Code, svcErr, reason, externalID)
}

// OrganisationNameInvalid indicates that OCM organisation was found, but its name is invalid.
func OrganisationNameInvalid(externalID string, name string) *ServiceError {
	return New(ErrorGeneral, "organisation with external id %q has invalid name %q", externalID, name)
}

// FailedClusterAuthorization converts the error to a ServiceError and returns a reason and hint for the user.
func FailedClusterAuthorization(err error) *ServiceError {
	svcErr := ToServiceError(err)
	reason := "failed to reserve quota"
	// Visiting the OpenShift page in console registers the user and their organisation with OCM.
	// See https://issues.redhat.com/browse/SDB-3194 for more context.
	if svcErr.Is404() {
		reason += " - visit https://console.redhat.com/openshift and try again"
	}
	return NewWithCause(svcErr.Code, svcErr, reason)
}
