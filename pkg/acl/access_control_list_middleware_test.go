package acl_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/server"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"

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
		dinosaur.ConfigProviders(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	env.MustResolveAll(&serverConfig)
	os.Exit(m.Run())
}

func Test_AccessControlListMiddleware_UserHasNoAccess(t *testing.T) {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, serverConfig.TokenIssuerURL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		arg  *acl.AccessControlListConfig
		want *errors.ServiceError
	}{
		{
			name: "returns 403 Forbidden response when user is not allowed to access service",
			arg: &acl.AccessControlListConfig{
				EnableDenyList: true,
				DenyList:       acl.DeniedUsers{"username"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)

			req, err := http.NewRequest("GET", "/api/dinosaurs_mgmt/dinosaurs", nil) // TODO change here to call your fleet manager endpoint
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			middleware := acl.NewAccessControlListMiddleware(tt.arg)
			handler := middleware.Authorize(http.HandlerFunc(NextHandler))

			// create a jwt and set it in the context
			ctx := req.Context()
			acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
			if err != nil {
				t.Fatal(err)
			}

			token, err := authHelper.CreateJWTWithClaims(acc, nil)
			if err != nil {
				t.Fatal(err)
			}

			ctx = auth.SetTokenInContext(ctx, token)
			req = req.WithContext(ctx)
			handler.ServeHTTP(rr, req)

			body, err := ioutil.ReadAll(rr.Body)
			if err != nil {
				t.Fatal(err)
			}

			Expect(rr.Code).To(Equal(http.StatusForbidden))
			Expect(rr.Header().Get("Content-Type")).To(Equal("application/json"))

			var data map[string]string
			err = json.Unmarshal(body, &data)
			if err != nil {
				t.Fatal(err)
			}
			Expect(data["kind"]).To(Equal("Error"))
			Expect(data["reason"]).To(Equal("User 'username' is not authorized to access the service."))
			// verify that context about user being allowed as service account is set to false always
			ctxAfterMiddleware := req.Context()
			Expect(auth.GetFilterByOrganisationFromContext(ctxAfterMiddleware)).To(Equal(false))
		})
	}
}

// NextHandler is a dummy handler that returns OK when QuotaList middleware has passed
func NextHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) //nolint
	_, err := io.WriteString(w, "OK")
	if err != nil {
		panic(err)
	}
}
