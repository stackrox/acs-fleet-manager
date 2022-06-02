package fleetmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	uri         = "api/rhacs/v1/agent-clusters"
	statusRoute = "status"

	publicCentralURI = "api/rhacs/v1/centrals"
)

// Client represents the client to REST client to connect to fleet-manager
type Client struct {
	client             http.Client
	ocmToken           string
	clusterID          string
	privateAPIEndpoint string
	publicAPIEndpoint  string
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
		return nil, errors.New("privateAPIEndpoint is empty")
	}

	return &Client{
		client:             http.Client{},
		clusterID:          clusterID,
		ocmToken:           ocmToken,
		privateAPIEndpoint: fmt.Sprintf("%s/%s/%s/%s", endpoint, uri, clusterID, "centrals"),
		publicAPIEndpoint:  fmt.Sprintf("%s/%s", endpoint, publicCentralURI),
	}, nil
}

// GetManagedCentralList returns a list of centrals from fleet-manager which should be managed by this fleetshard.
func (c *Client) GetManagedCentralList() (*private.ManagedCentralList, error) {
	resp, err := c.newRequest(http.MethodGet, c.privateAPIEndpoint, &bytes.Buffer{})
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	list := &private.ManagedCentralList{}
	err = json.Unmarshal(respBody, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateStatus batch updates the status of managed centrals. The status param takes a map of DataPlaneCentralStatus indexed by
// the Centrals ID.
func (c *Client) UpdateStatus(statuses map[string]private.DataPlaneCentralStatus) ([]byte, error) {
	updateBody, err := json.Marshal(statuses)
	if err != nil {
		return nil, err
	}

	bufUpdateBody := &bytes.Buffer{}
	_, err = bufUpdateBody.Write(updateBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", c.privateAPIEndpoint, statusRoute), bufUpdateBody)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}

func (c *Client) CreateCentral(request public.CentralRequestPayload) error {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return err
	}

	resp, err := c.newRequest(http.MethodPost, c.publicAPIEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println("%+v", all)
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
