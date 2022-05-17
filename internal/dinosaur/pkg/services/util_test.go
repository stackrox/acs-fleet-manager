package services

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	pkgErr "github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	serviceError "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"gorm.io/gorm"
)

const (
	resourceType           = "sampleResource"
	mockDinosaurRequestID  = "9bsv0s3fd06g02i2be9g" // sample dinosaur request ID
	mockIDWithInvalidChars = "vp&xG^nl9MStC@SI*#c$6V^TKq0"
)

func Test_HandleGetError(t *testing.T) {
	cause := pkgErr.WithStack(gorm.ErrInvalidData)
	type args struct {
		resourceType string
		field        string
		value        interface{}
		err          error
	}
	tests := []struct {
		name string
		args args
		want *serviceError.ServiceError
	}{
		{
			name: "Handler should return a general error for any errors other than record not found",
			args: args{
				resourceType: resourceType,
				field:        "id",
				value:        "sample-id",
				err:          cause,
			},
			want: serviceError.NewWithCause(serviceError.ErrorGeneral, cause, "Unable to find %s with id='sample-id'", resourceType),
		},
		{
			name: "Handler should return a not found error if record was not found in the database",
			args: args{
				resourceType: resourceType,
				field:        "id",
				value:        "sample-id",
				err:          gorm.ErrRecordNotFound,
			},
			want: serviceError.NotFound("%s with id='sample-id' not found", resourceType),
		},
		{
			name: "Handler should redact sensitive fields from the error message",
			args: args{
				resourceType: resourceType,
				field:        "email",
				value:        "sample@example.com",
				err:          gorm.ErrRecordNotFound,
			},
			want: serviceError.NotFound("%s with email='<redacted>' not found", resourceType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := services.HandleGetError(tt.args.resourceType, tt.args.field, tt.args.value, tt.args.err); !reflect.DeepEqual(got, tt.want) { //nolint:govet
				t.Errorf("HandleGetError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleCreateError(t *testing.T) {
	type args struct {
		resourceType string
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "Handler should return a general error for any other errors than violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          gorm.ErrInvalidField,
			},
			want: errors.GeneralError("Unable to create %s: %s", resourceType, gorm.ErrInvalidField.Error()),
		},
		{
			name: "Handler should return a conflict error if creation error is due to violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          fmt.Errorf("transaction violates unique constraints"),
			},
			want: errors.Conflict("This %s already exists", resourceType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := services.HandleCreateError(tt.args.resourceType, tt.args.err); !reflect.DeepEqual(got, tt.want) { //nolint:govet
				t.Errorf("handleCreateError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleUpdateError(t *testing.T) {
	type args struct {
		resourceType string
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "Handler should return a general error for any other errors than violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          gorm.ErrInvalidData,
			},
			want: errors.GeneralError("Unable to update %s: %s", resourceType, gorm.ErrInvalidData.Error()),
		},
		{
			name: "Handler should return a conflict error if update error is due to violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          fmt.Errorf("transaction violates unique constraints"),
			},
			want: errors.Conflict("Changes to %s conflict with existing records", resourceType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := services.HandleUpdateError(tt.args.resourceType, tt.args.err); !reflect.DeepEqual(got, tt.want) { //nolint:govet
				t.Errorf("handleUpdateError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_truncateString(t *testing.T) {
	exampleString := "example-string"
	type args struct {
		str string
		num int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should truncate string successfully",
			args: args{
				str: exampleString,
				num: 10,
			},
			want: exampleString[0:10],
		},
		{
			name: "should not truncate string if wanted length is less than given string length",
			args: args{
				str: exampleString,
				num: 15,
			},
			want: exampleString,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateString(tt.args.str, tt.args.num); got != tt.want {
				t.Errorf("truncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildTruncateDinosaurIdentifier(t *testing.T) {
	mockShortDinosaurName := "dinosaur"
	mockLongDinosaurName := "sample-dinosaur-name-long"

	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build dinosaur identifier with a short name successfully",
			args: args{
				dinosaurRequest: &dbapi.DinosaurRequest{
					Name: mockShortDinosaurName,
				},
			},
			want: fmt.Sprintf("%s-%s", mockShortDinosaurName, strings.ToLower(mockDinosaurRequestID)),
		},
		{
			name: "build dinosaur identifier with a long name successfully",
			args: args{
				dinosaurRequest: &dbapi.DinosaurRequest{
					Name: mockLongDinosaurName,
				},
			},
			want: fmt.Sprintf("%s-%s", mockLongDinosaurName[0:truncatedNameLen], strings.ToLower(mockDinosaurRequestID)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.dinosaurRequest.ID = mockDinosaurRequestID
			if got := buildTruncateDinosaurIdentifier(tt.args.dinosaurRequest); got != tt.want {
				t.Errorf("buildTruncateDinosaurIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_maskProceedingandTrailingDash(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should replace '-' prefix and suffix with a subdomain safe value",
			args: args{
				name: "-example-name-",
			},
			want: fmt.Sprintf("%[1]sexample-name%[1]s", appendChar),
		},
		{
			name: "should not replace '-' if its not a prefix or suffix of the given string",
			args: args{
				name: "example-name",
			},
			want: "example-name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskProceedingandTrailingDash(tt.args.name); got != tt.want {
				t.Errorf("maskProceedingandTrailingDash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_replaceHostSpecialChar(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "replace all invalid characters in an invalid host name",
			args: args{
				name: fmt.Sprintf("-host-%s", mockIDWithInvalidChars),
			},
			want: "ahost-vp-xg-nl-mstc-si-c--v-tkqa",
		},
		{
			name: "valid hostname should not be modified",
			args: args{
				name: "sample-host-name",
			},
			want: "sample-host-name",
		},
		{
			name: "should return an error if given host name is an empty string",
			args: args{
				name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceHostSpecialChar(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceHostSpecialChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("replaceHostSpecialChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	searchedString := "findMe"
	someSlice := []string{"some", "string", "values"}
	sliceWithFindMe := []string{"some", "string", "values", "findMe"}
	var emptySlice []string
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check for a string in an empty slice",
			args: args{
				s:     searchedString,
				slice: emptySlice,
			},
			want: false,
		},
		{
			name: "Check for a string in a non-empty slice that doesn't contain the string",
			args: args{
				s:     searchedString,
				slice: someSlice,
			},
			want: false,
		},
		{
			name: "Check for a string in a non-empty slice that contains that string",
			args: args{
				s:     searchedString,
				slice: sliceWithFindMe,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.Contains(tt.args.slice, tt.args.s)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
