package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigSuccess(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("SERVER_ADDRESS", ":8888")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("HTTPS_CERT_FILE", "/some/tls.crt")
	t.Setenv("HTTPS_KEY_FILE", "/some/tls.key")
	t.Setenv("METRICS_ADDRESS", ":9999")

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, cfg.ClusterID, "test-1")
	assert.Equal(t, cfg.ServerAddress, ":8888")
	assert.Equal(t, cfg.EnableHTTPS, true)
	assert.Equal(t, cfg.HTTPSCertFile, "/some/tls.crt")
	assert.Equal(t, cfg.HTTPSKeyFile, "/some/tls.key")
	assert.Equal(t, cfg.MetricsAddress, ":9999")
}

func TestGetConfigFailureMissingClusterID(t *testing.T) {
	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfigFailureEnabledHTTPSMissingCert(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("HTTPS_KEY_FILE", "/some/tls.key")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfigFailureEnabledHTTPSMissingKey(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("HTTPS_CERT_FILE", "/some/tls.crt")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfigFailureEnabledHTTPSOnly(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("ENABLE_HTTPS", "true")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// copied from a CRC openid-configuration response
const exampleOidcCfgContent = `{"issuer":"https://kubernetes.default.svc","jwks_uri":"https://api-int.crc.testing:6443/openid/v1/jwks","response_types_supported":["id_token"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"]}`

// copied from a CRC jwks response
const exampleJwksContent = `{"keys":[{"use":"sig","kty":"RSA","kid":"fopPsQkHnyVQN7buPdX_dZprJGWLS9yUB3snAklSwrU","alg":"RS256","n":"0xQ7zns3GOmClc5MLs4auWGrxndZnZ_UbzUC7gfhG2aIoUoJ7E8M5OVwl403nHo4mL8-7Q-U7xj59SFgLOfCCSbppW1VlaIec848RknnACSB-BArOKpoNliiSV5825P1ASgb2m5OJPdDTB6fe-7dSEXk_YjOVzuQUDB12b7oV6gjpKDspCAuK7jPiGyW_HrdavCPJu8zmHFmJUK8nhAE2eJy54BK4u7Iy6B8-al6Ah2ljxKrp_u6YQDyV_uXg4DjGM0iyZNOONmUdBrVKbnlUxtUYD-FIZgQxJad7qNX19dPt4yJE3DLZt4uA8A4GP6W-ZeI87AuvAVc0JXF_UJQiQ","e":"AQAB"},{"use":"sig","kty":"RSA","kid":"_xRpmmptK7pO0biiahy7FfW9msL0bOWSiUYHGfMBSjo","alg":"RS256","n":"wYWZZSARuDz1XCgaJ_MEG9znRz_9261tbZqsYpML2rioc41_8oRZE1QYKRUmHQnF51xkM6TuJr8lr10bP8mi17L_Y5UutQtCWuTKhwDwBfy3Bb-_dXwo9DCuK8gMryVcwViMWlOhFJ_573dpSfoQ3eyP3JkKMSFcn_aO5VVhJPvphDOj8cD8eJOilWCjjObpfNqHlcQUZ-rT15B6KzsVD_62SiecO6aEFU8jOYJGZOdCc4mp4ava7EW3jxbOAr3izTK781VS-PcuAQ1CxQA_H_Iwx1FMos70o0dxtFhZv0CNXQ9afATYha9vybksUaTkCStRI0hxnQaRoFzhsodiT0WH_JxsWBLrn38YWphAzTqMC8PtZYoaJnTzyme5Pq30xfr6Z_T-zk0TEB2PZ4jlw-2S2s3rxOPykaBf7tUOcrKzA0YZfn5LC_1DIt7B_IxGxjw5JMhz4-V15D-zOr0Mb0HWFnfhA1pNqNSGt9MdQMAVFGFP7PceKnthz0AqNI2u_J1f4KtuF_NCkmtyqlidevZD__QcKmobEpC81Zq05jNOLdDODpwQ9jdEMQKmlRZbVYdK8j0HcVhPZSMmWFMop94mwh9zuLkTr1xbv1Acnt9uUBaD6YYtC4TOSPAbKQUQfvZlwaeMp2RDclriafrEcbIq5P10oJFCs5mmjJH3KuM","e":"AQAB"},{"use":"sig","kty":"RSA","kid":"IXZasI0jKhRyIDobBVzn9WK28OFK0nf3csqvVacUYRw","alg":"RS256","n":"qZoqs8fW9RGNms1cTbTCRp9K1FNJDRPA16YcyyEBxMyA52g3lEtD7Qt59enBO4ecTj6E4_2qMQIvOSiq1scG5aROhgdG1ikXzFJP1oZiYBYUZ11tWtvH340mYNmucGQBjDOFtFZw8g-5JTir7PL2zdt1JtM2fyT9PIwXfsWtS9pedcMAJ0qFcv63JdTef3yxIbbpKPGjnOGZALSSP_GRpcXyUPGzByRZOcNcjYzcdU2bBed9x7pLz0ryv_E75mXnDN5FXi3oUI_WfMb_7s4ctV-RAe_KcFQVQ1O5CwUj7u6diRXZSZF_XiN09JPgOh1x8B_roviWr_ZrxA4uz9OcmxpzjTCJvO5V3L6S_WBwM4KhAdo2Cln1_oFdf_0FVx42iu9WPplNoBYDrHLxIXJXgykPQKsCBTiD9x6jHnDE60MENiVAg7MqaXvZ2JqtA7QDO8tUqhsPQTwNfJoowfofugKUhUpKl3KH8U5B-UY8Any3yvjtSv9uLxEUHMX-px-pPQGbtzcTgpyp2NNoxI_36HfFDX54HsEJRCUk4-E7S5XPu_e27iC4CFlvOIn7J6OZsnDIsLQFS2ff_uUx3ONIPiE3rDtFRo4ayE1iswCbJRHGIvYCBI3St5PbF9KCtKkBUsqoMEqqtgE06D5IsRCMZs6cgmAp9Reh93B9flTrh-8","e":"AQAB"}]}` // pragma: allowlist secret

func TestAuthConfigFromKubernetes(t *testing.T) {
	t.Setenv("AUTH_CONFIG_FROM_KUBERNETES", "true")
	testDir := t.TempDir()
	tokenFile := path.Join(testDir, "token")
	expectedToken := "faketoken"

	err := os.WriteFile(tokenFile, []byte(expectedToken), 0644)
	require.NoError(t, err, "failed to write test token to file")

	rt := fakeK8sRoundTripper{
		expectedToken: expectedToken,
		returnOidcCfg: exampleOidcCfgContent,
		returnJwks:    exampleJwksContent,
		t:             t,
	}
	fakeK8sClient := &http.Client{Transport: rt}

	authCfg := &AuthConfig{
		saTokenFile: tokenFile,
		httpClient:  fakeK8sClient,
		k8sJWKSPath: "/test/path",
		k8sSvcURL:   "localhost.testservice",
		jwksDir:     testDir,
	}

	err = authCfg.readFromKubernetes()
	require.NoError(t, err)
	require.Equal(t, []string{"https://kubernetes.default.svc"}, authCfg.AllowedIssuer, "issuers do not match")
	require.Equal(t, []string{"https://kubernetes.default.svc"}, authCfg.AllowedAudiences, "audiences do not match")
	require.Equal(t, []string{path.Join(testDir, "jwks.json")}, authCfg.JwksFiles, "jwks files do not match")
}

type fakeK8sRoundTripper struct {
	expectedToken string
	returnOidcCfg string
	returnJwks    string
	t             *testing.T
}

func (f fakeK8sRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token := req.Header.Get("Authorization")
	require.Equal(f.t, fmt.Sprintf("Bearer %s", f.expectedToken), token, "request token did not match expected token")

	res := &http.Response{}

	switch url := req.URL.String(); {
	case strings.Contains(url, wellKnownPath):
		res.StatusCode = 200
		res.Body = readCloserFromString(exampleOidcCfgContent)
	case strings.Contains(url, "/test/path"):
		res.StatusCode = 200
		res.Body = readCloserFromString(exampleJwksContent)
	default:
		res.StatusCode = 404
		res.Body = readCloserFromString("")
	}

	return res, nil
}

func readCloserFromString(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}
