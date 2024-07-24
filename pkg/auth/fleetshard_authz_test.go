package auth

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFleetShardAuthZConfig_ReadFiles(t *testing.T) {
	RegisterTestingT(t)
	c := &FleetShardAuthZConfig{
		Enabled: true,
		File:    "pkg/auth/testdata/fleetshard-authz.yaml",
	}
	err := c.ReadFiles()
	Expect(err).ToNot(HaveOccurred())
	Expect(c.AllowedSubjects).To(Equal(ClaimValues{"test-sub"}))
	Expect(c.AllowedAudiences).To(Equal(ClaimValues{"test-aud"}))
}
