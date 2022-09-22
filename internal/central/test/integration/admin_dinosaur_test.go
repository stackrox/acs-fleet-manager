package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	adminprivate "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/test"
	"github.com/stackrox/acs-fleet-manager/pkg/api"

	coreTest "github.com/stackrox/acs-fleet-manager/test"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	// TODO(ROX-9821) restore when admin API is properly implemented . "github.com/onsi/gomega"
)

func TestAdminCentral_Get(t *testing.T) {
	skipNotFullyImplementedYet(t)

	sampleCentralID := api.NewID()
	desiredCentralOperatorVersion := "test"
	type args struct {
		ctx       func(h *coreTest.Helper) context.Context
		centralID string
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result adminprivate.Central, resp *http.Response, err error)
	}{}
	/* TODO(ROX-9821) restore when admin API is properly implemented
	 {
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				centralID: sampleCentralID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should fail when the role defined in the request is not any of read, write or full",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{"notallowedrole"})
				},
				centralID: sampleCentralID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
				},
				centralID: sampleCentralID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleCentralID))
				Expect(result.DesiredCentralOperatorVersion).To(Equal(desiredCentralOperatorVersion))
				Expect(result.AccountNumber).ToNot(BeEmpty())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminWriteRole})
				},
				centralID: sampleCentralID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleCentralID))
				Expect(result.DesiredCentralOperatorVersion).To(Equal(desiredCentralOperatorVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				centralID: sampleCentralID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleCentralID))
				Expect(result.DesiredCentralOperatorVersion).To(Equal(desiredCentralOperatorVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: "should fail when the requested dinosaur does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
				},
				centralID: "unexistingdinosaurID",
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should fail when the request does not contain a valid issuer",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					account := h.NewAllowedServiceAccount()
					claims := jwt.MapClaims{
						"iss": "invalidiss",
						"realm_access": map[string][]string{
							"roles": {auth.FleetManagerAdminReadRole},
						},
					}
					token := h.CreateJWTStringWithClaim(account, claims)
					ctx := context.WithValue(context.Background(), adminprivate.ContextAccessToken, token)
					return ctx
				},
				centralID: sampleCentralID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
	} */

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewCentralHelper(t, ocmServer)
	defer tearDown()
	db := test.TestServices.DBFactory.New()
	dinosaur := &dbapi.CentralRequest{
		MultiAZ:                       false,
		Owner:                         "test-user",
		Region:                        "test",
		CloudProvider:                 "test",
		Name:                          "test-dinosaur",
		OrganisationID:                "13640203",
		DesiredCentralOperatorVersion: desiredCentralOperatorVersion,
		Status:                        constants.CentralRequestStatusReady.String(),
		Namespace:                     fmt.Sprintf("dinosaur-%s", sampleCentralID),
	}
	dinosaur.ID = sampleCentralID

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.GetCentralById(ctx, tt.args.centralID)
			tt.verifyResponse(result, resp, err)
		})
	}
}
