package authorization

import (
	"fmt"
	"net/http"

	sdkClient "github.com/openshift-online/ocm-sdk-go"
)

// Authorization ...
//go:generate moq -out authorization_moq.go . Authorization
type Authorization interface {
	CheckUserValid(username string, orgID string) (bool, error)
}

type authorization struct {
	client *sdkClient.Connection
}

var _ Authorization = &authorization{}

// NewOCMAuthorization ...
func NewOCMAuthorization(client *sdkClient.Connection) Authorization {
	return &authorization{
		client: client,
	}
}

// CheckUserValid ...
func (a authorization) CheckUserValid(username string, orgID string) (bool, error) {
	resp, err := a.client.AccountsMgmt().V1().Accounts().List().
		Parameter("page", 1).
		Parameter("size", 1).
		Parameter("search", fmt.Sprintf("username = '%s'", username)).
		Send()

	return resp.Status() == http.StatusOK && resp.Size() > 0 && !resp.Items().Get(0).Banned() &&
		resp.Items().Get(0).Organization().ExternalID() == orgID, err
}
