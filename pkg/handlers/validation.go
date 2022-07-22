package handlers

import (
	"net/http"
	"regexp"

	"github.com/xeipuuv/gojsonschema"

	"net/url"
	"strconv"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// ValidUUIDRegexp ...
var (
	// Dinosaur cluster names must consist of lower-case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character. For example, 'my-name', or 'abc-123'.

	ValidUUIDRegexp               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	ValidServiceAccountNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	ValidServiceAccountDescRegexp = regexp.MustCompile(`^[a-zA-Z0-9.,\-\s]*$`)
	MinRequiredFieldLength        = 1

	MaxServiceAccountNameLength = 50
	MaxServiceAccountDescLength = 255
	MaxServiceAccountID         = 36
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

// ValidateServiceAccountName ...
func ValidateServiceAccountName(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidServiceAccountNameRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountName("%s does not match %s", field, ValidServiceAccountNameRegexp.String())
		}
		return nil
	}
}

// ValidateServiceAccountDesc ...
func ValidateServiceAccountDesc(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidServiceAccountDescRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountDesc("%s does not match %s", field, ValidServiceAccountDescRegexp.String())
		}
		return nil
	}
}

// ValidateServiceAccountID ...
func ValidateServiceAccountID(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidUUIDRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountID("%s does not match %s", field, ValidUUIDRegexp.String())
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

// ValidateJSONSchema ...
func ValidateJSONSchema(schemaName string, schemaLoader gojsonschema.JSONLoader, documentName string, documentLoader gojsonschema.JSONLoader) *errors.ServiceError {
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return errors.BadRequest("invalid %s: %v", schemaName, err)
	}

	r, err := schema.Validate(documentLoader)
	if err != nil {
		return errors.BadRequest("invalid %s: %v", documentName, err)
	}
	if !r.Valid() {
		return errors.BadRequest("%s not conform to the %s. %d errors encountered.  1st error: %s",
			documentName, schemaName, len(r.Errors()), r.Errors()[0].String())
	}
	return nil
}

// ValidatQueryParam ...
func ValidatQueryParam(queryParams url.Values, field string) Validate {

	return func() *errors.ServiceError {
		fieldValue := queryParams.Get(field)
		if fieldValue == "" {
			return errors.FailedToParseQueryParms("bad request, cannot parse query parameter '%s' '%s'", field, fieldValue)
		}
		if _, err := strconv.ParseInt(fieldValue, 10, 64); err != nil {
			return errors.FailedToParseQueryParms("bad request, cannot parse query parameter '%s' '%s'", field, fieldValue)
		}

		return nil
	}

}
