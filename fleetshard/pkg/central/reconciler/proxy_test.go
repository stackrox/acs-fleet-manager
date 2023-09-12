package reconciler

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"
)

const testNS = `acscs-01`

func testProxyConfiguration(t *testing.T, noProxyURLs []string, proxiedURLs []string) {
	proxyFunc := httpproxy.FromEnvironment().ProxyFunc()

	for _, u := range noProxyURLs {
		parsedURL, err := url.Parse(u)
		require.NoError(t, err)

		proxyURL, err := proxyFunc(parsedURL)
		require.NoError(t, err)
		assert.Nilf(t, proxyURL, "expected URL %s to not be proxied, got: %s", u, proxyURL)
	}

	const expectedProxyURL = "http://egress-proxy.acscs-01.svc:3128"

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

func TestProxyConfiguration(t *testing.T) {
	for _, envVar := range getProxyEnvVars(testNS) {
		t.Setenv(envVar.Name, envVar.Value)
	}

	noProxyURLs := []string{
		"https://central",
		"https://central.acscs-01",
		"https://central.acscs-01.svc",
		"https://central.acscs-01.svc:443",
		"https://scanner-db.acscs-01.svc:5432",
		"https://scanner:8443",
		"https://scanner.acscs-01:8080",
	}

	proxiedURLs := []string{
		"https://audit-logs-aggregator.rhacs-audit-logs:8888",
		"https://www.example.com",
		"https://www.example.com:8443",
		"http://example.com",
		"http://example.com:8080",
		"https://central.acscs-01.svc:8443",
		"https://scanner.acscs-01.svc",
	}

	testProxyConfiguration(t, noProxyURLs, proxiedURLs)
}

func TestProxyConfiguration_IsDeterministic(t *testing.T) {
	envVars := getProxyEnvVars(testNS)
	for i := 0; i < 5; i++ {
		otherEnvVars := getProxyEnvVars(testNS)
		assert.Equal(t, envVars, otherEnvVars)
	}
}

var (
	additionalNoProxyURLs = []url.URL{
		{
			Host: "audit-logs-aggregator.rhacs-audit-logs:8888",
		},
	}
)

func TestProxyConfigurationWithAdditionalDirectAccess(t *testing.T) {
	for _, envVar := range getProxyEnvVars(testNS, additionalNoProxyURLs...) {
		t.Setenv(envVar.Name, envVar.Value)
	}

	noProxyURLs := []string{
		"https://central",
		"https://central.acscs-01",
		"https://central.acscs-01.svc",
		"https://central.acscs-01.svc:443",
		"https://scanner-db.acscs-01.svc:5432",
		"https://scanner:8443",
		"https://scanner.acscs-01:8080",
		"https://audit-logs-aggregator.rhacs-audit-logs:8888",
	}

	proxiedURLs := []string{
		"https://www.example.com",
		"https://www.example.com:8443",
		"http://example.com",
		"http://example.com:8080",
		"https://central.acscs-01.svc:8443",
		"https://scanner.acscs-01.svc",
	}

	testProxyConfiguration(t, noProxyURLs, proxiedURLs)
}

func TestProxyConfigurationWithAdditionalDirectAccess_IsDeterministic(t *testing.T) {
	envVars := getProxyEnvVars(testNS, additionalNoProxyURLs...)
	for i := 0; i < 5; i++ {
		otherEnvVars := getProxyEnvVars(testNS, additionalNoProxyURLs...)
		assert.Equal(t, envVars, otherEnvVars)
	}
}
