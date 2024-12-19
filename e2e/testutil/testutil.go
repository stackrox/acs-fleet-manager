// Package testutil implements utility routines used in ACSCS e2e tests
package testutil

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

const defaultTimeout = 5 * time.Minute

var (
	// SkipDNSMsg is the message printed when DNS e2e tests or assertions should be skipped
	SkipDNSMsg = "external DNS is not enabled for this test run"
)

// GetWaitTimeout gets the test wait timeout for polling operation from
// OS environment WAIT_TIMEOUT or returns the defaultTimeout if unset
func GetWaitTimeout() time.Duration {
	timeoutStr, ok := os.LookupEnv("WAIT_TIMEOUT")
	if ok {
		timeout, err := time.ParseDuration(timeoutStr)
		if err == nil {
			return timeout
		}
		fmt.Printf("Error parsing timeout, using default timeout %v: %s\n", defaultTimeout, err)
	}
	return defaultTimeout
}

// DNSConfiguration looks for propper environment variable setup to run e2e tests
// with Route53 DNS functionality enabled and returns it.
func DNSConfiguration(routesEnabled bool) (dnsEnabled bool, accessKey string, secretKey string) {
	accessKey = os.Getenv("ROUTE53_ACCESS_KEY")
	secretKey = os.Getenv("ROUTE53_SECRET_ACCESS_KEY")
	enableExternal := os.Getenv("ENABLE_CENTRAL_EXTERNAL_CERTIFICATE")
	dnsEnabled = accessKey != "" &&
		secretKey != "" &&
		enableExternal != "" && routesEnabled
	return dnsEnabled, accessKey, secretKey
}

// SkipIf skips a Gingko test container if condition is true
func SkipIf(condition bool, message string) {
	if condition {
		Skip(message, 1)
	}
}

// GetCentralRequest queries fleet-manager public API for the CentralRequest with id and stores in in the given pointer
func GetCentralRequest(ctx context.Context, client *fleetmanager.Client, id string, request *public.CentralRequest) error {
	centralRequest, _, err := client.PublicAPI().GetCentralById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to obtain CentralRequest: %w", err)
	}
	*request = centralRequest
	return nil
}
