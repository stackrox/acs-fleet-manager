package services

import (
	"strings"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"gorm.io/gorm"
)

// Field names suspected to contain personally identifiable information
var piiFields = []string{
	"username",
	"first_name",
	"last_name",
	"email",
	"address",
}

// HandleGetError ...
func HandleGetError(resourceType, field string, value interface{}, err error) *errors.ServiceError {
	// Sanitize errors of any personally identifiable information
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	if IsRecordNotFoundError(err) {
		return errors.NotFound("%s with %s='%v' not found", resourceType, field, value)
	}
	return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find %s with %s='%v'", resourceType, field, value)
}

// HandleGoneError ...
func HandleGoneError(resourceType, field string, value interface{}) *errors.ServiceError {
	// Sanitize errors of any personally identifiable information
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	return errors.New(errors.ErrorGone, "%s with %s='%v' has been deleted", resourceType, field, value)
}

// HandleDeleteError ...
func HandleDeleteError(resourceType string, field string, value interface{}, err error) *errors.ServiceError {
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to delete %s with %s='%v'", resourceType, field, value)
}

// IsRecordNotFoundError ...
func IsRecordNotFoundError(err error) bool {
	return err == gorm.ErrRecordNotFound
}

// HandleCreateError ...
func HandleCreateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("This %s already exists", resourceType)
	}
	return errors.GeneralError("Unable to create %s: %s", resourceType, err.Error())
}

// HandleUpdateError ...
func HandleUpdateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("Changes to %s conflict with existing records", resourceType)
	}
	return errors.GeneralError("Unable to update %s: %s", resourceType, err.Error())
}
