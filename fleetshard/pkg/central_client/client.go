package central_client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
)

func SendRequestToCentral(ctx context.Context, requestBody interface{}, address string, pass string) (*http.Response, error) {
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling new request to central")
	}
	req, err := http.NewRequest(http.MethodPost, address, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, errors.Wrap(err, "creating HTTP request to central")
	}
	req.SetBasicAuth("admin", pass)
	req = req.WithContext(ctx)

	insecureTransport := http.DefaultTransport.(*http.Transport).Clone()
	// TODO: once certificates will be added, we probably will be able to replace with secure transport
	insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{
		Transport: insecureTransport,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending new request to central")
	}
	return resp, nil
}
