package handlers

import (
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// ValidUUIDRegexp ...
var (
	MinRequiredFieldLength      = 1
	MaxServiceAccountDescLength = 255
)

// ValidateAsyncEnabled returns a validator that returns an error if the async query param is not true
func ValidateAsyncEnabled(r *http.Request, action string) Validate {
	return func() *errors.ServiceError {
		asyncParam := r.URL.Query().Get("async")
		if asyncParam != "true" {
			return errors.SyncActionNotSupported()
		}
		return nil
	}
}

// ValidateMultiAZEnabled returns a validator that returns an error if the multiAZ is not true
func ValidateMultiAZEnabled(value *bool, action string) Validate {
	return func() *errors.ServiceError {
		if !*value {
			return errors.NotMultiAzActionNotSupported()
		}
		return nil
	}
}

// ValidateMaxLength ...
func ValidateMaxLength(value *string, field string, maxVal *int) Validate {
	return func() *errors.ServiceError {
		if maxVal != nil && len(*value) > *maxVal {
			return errors.MaximumFieldLengthMissing("%s is not valid. Maximum length %d is required", field, *maxVal)
		}
		return nil
	}
}

// ValidateLength ...
func ValidateLength(value *string, field string, minVal *int, maxVal *int) Validate {
	var min = 1
	if *minVal > 1 {
		min = *minVal
	}
	resp := ValidateMaxLength(value, field, maxVal)
	if resp != nil {
		return resp
	}
	return ValidateMinLength(value, field, min)
}

// ValidateMinLength ...
func ValidateMinLength(value *string, field string, min int) Validate {
	return func() *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, min)
		}
		return nil
	}
}

func ValidateRegex(r *http.Request, field string, regex *regexp.Regexp) Validate {
	return func() *errors.ServiceError {
		value := mux.Vars(r)[field]
		if !regex.MatchString(value) {
			return errors.MalformedServiceAccountName("%s %q does not match %s", field, value, regex.String())
		}
		return nil
	}
}
