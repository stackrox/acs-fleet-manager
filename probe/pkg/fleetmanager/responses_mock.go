package fleetmanager

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

func makeHTTPResponse(statusCode int) *http.Response {
	json := `{}`
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	response := &http.Response{
		StatusCode: statusCode,
		Body:       r,
	}
	return response
}

// Predefined responses.
var (
	CreateCentralResponseAccepted = &CreateCentralResponse{
		Response: makeHTTPResponse(http.StatusAccepted),
	}
	CreateCentralResponseStatusServerError = &CreateCentralResponse{
		Response: makeHTTPResponse(http.StatusInternalServerError),
	}
	CreateCentralResponseError = &CreateCentralResponse{
		Response: makeHTTPResponse(http.StatusAccepted),
		Err:      errors.New("Failed response"),
	}

	DeleteCentralByIDResponseAccepted          = &DeleteCentralByIDResponse{Response: makeHTTPResponse(http.StatusAccepted)}
	DeleteCentralByIDResponseStatusServerError = &DeleteCentralByIDResponse{Response: makeHTTPResponse(http.StatusInternalServerError)}
	DeleteCentralByIDResponseError             = &DeleteCentralByIDResponse{
		Response: makeHTTPResponse(http.StatusAccepted),
		Err:      errors.New("Failed response"),
	}

	GetCentralByIDResponseAccepted = &GetCentralByIDResponse{
		Request:  public.CentralRequest{InstanceType: StandardInstanceType, Status: constants.CentralRequestStatusAccepted.String()},
		Response: makeHTTPResponse(http.StatusOK),
	}
	GetCentralByIDResponseReady = &GetCentralByIDResponse{
		Request:  public.CentralRequest{InstanceType: StandardInstanceType, Status: constants.CentralRequestStatusReady.String()},
		Response: makeHTTPResponse(http.StatusOK),
	}
	GetCentralByIDResponseDeprovision = &GetCentralByIDResponse{
		Request:  public.CentralRequest{InstanceType: StandardInstanceType, Status: constants.CentralRequestStatusDeprovision.String()},
		Response: makeHTTPResponse(http.StatusOK),
	}
	GetCentralByIDResponseStatusServerError = &GetCentralByIDResponse{
		Request:  public.CentralRequest{InstanceType: StandardInstanceType, Status: constants.CentralRequestStatusAccepted.String()},
		Response: makeHTTPResponse(http.StatusInternalServerError),
	}
	GetCentralByIDResponseError = &GetCentralByIDResponse{
		Request:  public.CentralRequest{InstanceType: StandardInstanceType, Status: constants.CentralRequestStatusDeprovision.String()},
		Response: makeHTTPResponse(http.StatusOK),
		Err:      errors.New("Failed response"),
	}
)
