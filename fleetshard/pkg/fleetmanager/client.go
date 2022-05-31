package fleetmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	uri         = "api/rhacs/v1/agent-clusters"
	statusRoute = "status"
)

type Client struct {
	client    http.Client
	ocmToken  string
	clusterID string
	endpoint  string
}

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
		endpoint:  fmt.Sprintf("%s/%s/%s/%s", endpoint, uri, clusterID, "dinosaurs"),
	}, nil
}

func (c *Client) GetManagedCentralList() (*private.ManagedCentralList, error) {
	resp, err := c.newRequest(http.MethodGet, c.endpoint, &bytes.Buffer{})
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

	resp, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", c.endpoint, statusRoute), bufUpdateBody)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}

func (c *Client) newRequest(method string, url string, body io.Reader) (*http.Response, error) {
	glog.Infof("Send request to %s;", url)
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
