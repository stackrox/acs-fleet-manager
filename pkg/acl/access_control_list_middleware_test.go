package acl_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/central"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/server"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"

	. "github.com/onsi/gomega"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

var env *environments.Env
var serverConfig *server.ServerConfig

func TestMain(m *testing.M) {
	var err error
	env, err = environments.New(environments.GetEnvironmentStrFromEnv(),
		central.ConfigProviders(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	env.MustResolveAll(&serverConfig)
	os.Exit(m.Run())
}

func Test_AccessControlListMiddleware_UserHasNoAccess(t *testing.T) {
	RegisterTestingT(t)
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, serverConfig.TokenIssuerURL)
	Expect(err).NotTo(HaveOccurred())

	tests := []struct {
		name           string
		arg            *acl.AccessControlListConfig
		wantErr        bool
		wantHTTPStatus int
	}{
		{
			name: "returns 403 Forbidden response when user is not allowed to access service",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: true,
				DenyList:       acl.DeniedUsers{"username"},
			},
			wantErr:        true,
			wantHTTPStatus: http.StatusForbidden,
		},
		{
			name: "returns 200 status if denyList is disabled",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: false,
			},
			wantErr:        false,
			wantHTTPStatus: http.StatusOK,
		},
		{
			name: "returns 200 status if denyList is enabled and deny list is empty",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: true,
			},
			wantErr:        false,
			wantHTTPStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/rhacs/centrals", nil) // TODO change here to call your fleet manager endpoint
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()

			middleware := acl.NewAccessControlListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			// create a jwt and set it in the context
			ctx := req.Context()
			acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
			Expect(err).NotTo(HaveOccurred())

			token, err := authHelper.CreateJWTWithClaims(acc, nil)
			Expect(err).NotTo(HaveOccurred())

			ctx = auth.SetTokenInContext(ctx, token)
			req = req.WithContext(ctx)
			handler.ServeHTTP(rr, req)

			body, err := ioutil.ReadAll(rr.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Code).To(Equal(tt.wantHTTPStatus))

			if tt.wantErr {
				Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))
				var data map[string]string
				err = json.Unmarshal(body, &data)
				Expect(err).NotTo(HaveOccurred())
				Expect(data["kind"]).To(Equal("Error"))
				Expect(data["reason"]).To(Equal("User \"username\" is not authorized to access the service."))
				// verify that context about user being allowed as service account is set to false always
				ctxAfterMiddleware := req.Context()
				Expect(auth.GetFilterByOrganisationFromContext(ctxAfterMiddleware)).To(Equal(false))
			}
		})
	}
}

// NextHandler is a dummy handler that returns OK when QuotaList middleware has passed
func NextHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "OK")
	Expect(err).NotTo(HaveOccurred())
}
