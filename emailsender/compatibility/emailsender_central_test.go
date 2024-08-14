//go:build test_central_compatibility

package compatibility

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	centralURLFmt    = "https://%s:%s@localhost:8443" // pragma: allowlist secret
	testNotifierPath = "/v1/notifiers/test"
	adminUser        = "admin"
)

//go:embed post-acscsemail-integration.json
var notifierPayload []byte

func TestACSCSEmailNotifier(t *testing.T) {
	adminPW := os.Getenv("ADMIN_PW")
	require.NotEmpty(t, adminPW, "AMDIN_PW is not set in environment")

	url := fmt.Sprintf(centralURLFmt, adminUser, adminPW)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(notifierPayload))
	require.NoError(t, err, "failed to build http request")

	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	res, err := httpClient.Do(req)
	require.NoError(t, err, "failed to send notifier test requests to central")
	defer res.Body.Close()

	status := res.StatusCode
	if status == 200 {
		return
	}

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "failed to read response body of response")
	t.Fatalf("requests has status code: %d, with body: %s", status, string(resBody))
}
