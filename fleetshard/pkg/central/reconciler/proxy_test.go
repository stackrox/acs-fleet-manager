package reconciler

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"
)

const testNS = `acsms-01`

func TestProxyConfiguration(t *testing.T) {
	for _, envVar := range getProxyEnvVars(testNS) {
		t.Setenv(envVar.Name, envVar.Value)
	}

	proxyFunc := httpproxy.FromEnvironment().ProxyFunc()

	noProxyURLs := []string{
		"https://central",
		"https://central.acsms-01",
		"https://central.acsms-01.svc",
		"https://central.acsms-01.svc:443",
		"https://scanner-db.acsms-01.svc:5432",
		"https://scanner:8443",
		"https://scanner.acsms-01:8080",
	}

	for _, u := range noProxyURLs {
		parsedURL, err := url.Parse(u)
		require.NoError(t, err)

		proxyURL, err := proxyFunc(parsedURL)
		require.NoError(t, err)
		assert.Nilf(t, proxyURL, "expected URL %s to not be proxied, got: %s", u, proxyURL)
	}

	proxiedURLs := []string{
		"https://www.example.com",
		"https://www.example.com:8443",
		"http://example.com",
		"http://example.com:8080",
		"https://central.acsms-01.svc:8443",
		"https://scanner.acsms-01.svc",
	}
	const expectedProxyURL = "http://egress-proxy.acsms-01.svc:3128"

	for _, u := range proxiedURLs {
		parsedURL, err := url.Parse(u)
		require.NoError(t, err)

		proxyURL, err := proxyFunc(parsedURL)
		require.NoError(t, err)
		if !assert.NotNilf(t, proxyURL, "expected URL %s to be proxied", u) {
			continue
		}
		assert.Equal(t, expectedProxyURL, proxyURL.String())
	}
}

func TestProxyConfiguration_IsDeterministic(t *testing.T) {
	envVars := getProxyEnvVars(testNS)
	for i := 0; i < 5; i++ {
		otherEnvVars := getProxyEnvVars(testNS)
		assert.Equal(t, envVars, otherEnvVars)
	}
}
