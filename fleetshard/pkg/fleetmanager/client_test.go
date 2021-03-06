package fleetmanager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noAuth struct{}

func (n noAuth) AddAuth(_ *http.Request) error {
	return nil
}

func TestClientGetManagedCentralList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Contains(t, request.RequestURI, "/api/rhacs/v1/agent-clusters/cluster-id/centrals")
		bytes, err := json.Marshal(private.ManagedCentralList{})
		require.NoError(t, err)
		_, err = writer.Write(bytes)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, "cluster-id", &noAuth{})
	require.NoError(t, err)

	result, err := client.GetManagedCentralList()
	require.NoError(t, err)
	assert.Equal(t, &private.ManagedCentralList{}, result)
}

func TestClientReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Contains(t, request.RequestURI, "/api/rhacs/v1/agent-clusters/cluster-id/centrals")
		bytes, err := json.Marshal(compat.Error{
			Kind:   "error",
			Reason: "some reason",
		})
		require.NoError(t, err)
		_, err = writer.Write(bytes)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, "cluster-id", &noAuth{})
	require.NoError(t, err)

	_, err = client.GetManagedCentralList()
	require.Error(t, err)
	assert.ErrorContains(t, err, "some reason")
}

func TestClientUpdateStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Contains(t, request.RequestURI, "/api/rhacs/v1/agent-clusters/cluster-id/centrals")
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, "cluster-id", &noAuth{})
	require.NoError(t, err)

	statuses := map[string]private.DataPlaneCentralStatus{}
	err = client.UpdateStatus(statuses)
	require.NoError(t, err)
}
