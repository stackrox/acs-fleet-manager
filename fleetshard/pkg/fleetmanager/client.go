package fleetmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"io"
	"net/http"
	"os"
)

const (
	uri         = "api/rhacs/v1/agent-clusters"
	statusRoute = "status"
)

// Client represents the client to REST client to connect to fleet-manager
type Client struct {
	client    http.Client
	ocmToken  string
	clusterID string
	endpoint  string
}

// NewClient creates a new client
func NewClient(endpoint string, clusterID string) (*Client, error) {
	//TODO(create-ticket): Add authentication SSO
	ocmToken := os.Getenv("OCM_TOKEN")
	if ocmToken == "" {
		return nil, errors.New("empty ocm token")
	}

	if clusterID == "" {
		return nil, errors.New("cluster id is empty")
	}

	if endpoint == "" {
		return nil, errors.New("endpoint is empty")
	}

	return &Client{
		client:    http.Client{},
		clusterID: clusterID,
		ocmToken:  ocmToken,
		endpoint:  fmt.Sprintf("%s/%s/%s/%s", endpoint, uri, clusterID, "centrals"),
	}, nil
}

// GetManagedCentralList returns a list of centrals from fleet-manager which should be managed by this fleetshard.
func (c *Client) GetManagedCentralList() (*private.ManagedCentralList, error) {
	resp, err := c.newRequest(http.MethodGet, c.endpoint, &bytes.Buffer{})
	if err != nil {
		return nil, err
	}

	list := &private.ManagedCentralList{}
	err = c.unmarshalResponse(resp.Body, &list)
	if err != nil {
		return nil, errors.Wrapf(err, "calling %s", c.endpoint)
	}

	return list, nil
}

// UpdateStatus batch updates the status of managed centrals. The status param takes a map of DataPlaneCentralStatus indexed by
// the Centrals ID.
func (c *Client) UpdateStatus(statuses map[string]private.DataPlaneCentralStatus) error {
	updateBody, err := json.Marshal(statuses)
	if err != nil {
		return err
	}

	bufUpdateBody := &bytes.Buffer{}
	_, err = bufUpdateBody.Write(updateBody)
	if err != nil {
		return err
	}

	resp, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", c.endpoint, statusRoute), bufUpdateBody)
	if err != nil {
		return err
	}

	if err := c.unmarshalResponse(resp.Body, &struct{}{}); err != nil {
		return errors.Wrapf(err, "updating status")
	}
	return nil
}

func (c *Client) newRequest(method string, url string, body io.Reader) (*http.Response, error) {
	glog.Infof("Send request to %s", url)
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.ocmToken))

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) unmarshalResponse(body io.Reader, v interface{}) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	into := struct {
		Kind string `json:"kind"`
	}{}
	err = json.Unmarshal(data, &into)
	if err != nil {
		return err
	}

	// Unmarshal error
	if into.Kind == "error" {
		apiError := compat.Error{}
		err = json.Unmarshal(data, &apiError)
		if err != nil {
			return err
		}
		return errors.Errorf("API error occured %s: %s", apiError.Code, apiError.Reason)
	}

	return json.Unmarshal(data, v)
}
