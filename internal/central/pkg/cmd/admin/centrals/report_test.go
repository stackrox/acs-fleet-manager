package centrals

import (
	"bytes"
	"testing"
	"time"

	adminAPI "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stretchr/testify/assert"
)

func newCentral(id, name, orgID, orgName, owner, region, instanceType, quotaType string) adminAPI.Central {
	return adminAPI.Central{
		Id:               id,
		Name:             name,
		OrganisationId:   orgID,
		OrganisationName: orgName,
		Owner:            owner,
		Region:           region,
		InstanceType:     instanceType,
		QuotaType:        quotaType,
		Status:           "ready",
		CreatedAt:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestRegionRow(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")

	got := regionRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "2026-07-01", "standard/ams", "ready"}, got)
}

func TestRegionRow_Provisioning(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")
	c.Status = "provisioning"

	got := regionRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "2026-07-01", "standard/ams", "provisioning"}, got)
}

func TestFullRow(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")

	got := fullRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "2026-07-01", "us-east-1", "standard/ams", "ready"}, got)
}

func TestExpiredRow_WithExpiration(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")
	expiry := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	c.ExpiredAt = &expiry

	got := expiredRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "2026-06-15 14:30", "us-east-1", "ready"}, got)
}

func TestExpiredRow_WithoutExpiration(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")

	got := expiredRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "-", "us-east-1", "ready"}, got)
}

func TestOrgColumn_NoOrgId(t *testing.T) {
	c := newCentral("abc123", "my-central", "", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")

	got := orgColumn(&c)

	assert.Equal(t, "Acme Corp", got)
}

func TestOrgColumn_NoOrgName(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "", "jdoe", "us-east-1", "standard", "ams")

	got := orgColumn(&c)

	assert.Equal(t, "- (12345)", got)
}

func TestOrgColumn_NoBoth(t *testing.T) {
	c := newCentral("abc123", "my-central", "", "", "jdoe", "us-east-1", "standard", "ams")

	got := orgColumn(&c)

	assert.Equal(t, "-", got)
}

func TestTypeQuota_NoQuota(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "")

	got := typeQuota(&c)

	assert.Equal(t, "standard", got)
}

func TestWriteSection_Empty(t *testing.T) {
	var b bytes.Buffer
	writeSection(&b, "Test Section", nil, []string{"ID", "Name"}, fullRow)

	assert.Equal(t, "\nTest Section [0]\n(none)\n", b.String())
}

func TestWriteSection_WithInstances(t *testing.T) {
	instances := []adminAPI.Central{
		newCentral("id1", "central-1", "100", "Acme", "alice", "us-east-1", "standard", "ams"),
		newCentral("id2", "central-2", "200", "Globex", "bob", "eu-west-1", "eval", "quota-management-list"),
	}

	var b bytes.Buffer
	headers := []string{"ID", "Name", "Org", "Owner", "Created", "Region", "Type/Quota", "Status"}
	writeSection(&b, "My Instances", instances, headers, fullRow)

	got := b.String()
	assert.Contains(t, got, "My Instances [2]")
	assert.Contains(t, got, "```")
	assert.Contains(t, got, "ID")
	assert.Contains(t, got, "| Name")
	assert.Contains(t, got, "| Org")
	assert.Contains(t, got, "id1")
	assert.Contains(t, got, "| central-1")
	assert.Contains(t, got, "| Acme (100)")
	assert.Contains(t, got, "| alice")
	assert.Contains(t, got, "id2")
	assert.Contains(t, got, "| central-2")
	assert.Contains(t, got, "| Globex (200)")
	assert.Contains(t, got, "| bob")
	assert.NotContains(t, got, "(none)")
}

func TestWriteSection_WithExpiredFormatter(t *testing.T) {
	c := newCentral("id1", "central-1", "100", "Acme", "alice", "us-east-1", "standard", "ams")
	expiry := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	c.ExpiredAt = &expiry

	var b bytes.Buffer
	headers := []string{"ID", "Name", "Org", "Owner", "Expires", "Region", "Status"}
	writeSection(&b, "Expired", []adminAPI.Central{c}, headers, expiredRow)

	got := b.String()
	assert.Contains(t, got, "Expired [1]")
	assert.Contains(t, got, "| Org")
	assert.Contains(t, got, "| Expires")
	assert.Contains(t, got, "| 2026-05-31 00:00")
}

func TestWriteSection_ColumnsAligned(t *testing.T) {
	instances := []adminAPI.Central{
		newCentral("short", "a", "1", "O", "x", "us-east-1", "s", "q"),
		newCentral("much-longer-id", "longer-name", "99999", "OrgLong", "longowner", "eu-west-1", "standard", "ams"),
	}

	var b bytes.Buffer
	headers := []string{"ID", "Name", "Org ID"}
	writeSection(&b, "Aligned", instances, headers, func(c *adminAPI.Central) []string {
		return []string{c.Id, c.Name, c.OrganisationId}
	})

	got := b.String()
	assert.Contains(t, got, "short           | a            | 1")
	assert.Contains(t, got, "much-longer-id  | longer-name  | 99999")
}

func TestFailedRow_WithReason(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")
	c.FailedReason = "cluster capacity exceeded"

	got := failedRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "us-east-1", "standard/ams", "cluster capacity exceeded"}, got)
}

func TestFailedRow_WithoutReason(t *testing.T) {
	c := newCentral("abc123", "my-central", "12345", "Acme Corp", "jdoe", "us-east-1", "standard", "ams")

	got := failedRow(&c)

	assert.Equal(t, []string{"abc123", "my-central", "Acme Corp (12345)", "jdoe", "us-east-1", "standard/ams", "-"}, got)
}

func TestBuildStatusFilter(t *testing.T) {
	filter := buildStatusFilter()

	assert.Contains(t, filter, "status <> failed")
	assert.NotContains(t, filter, "deprovision")
	assert.NotContains(t, filter, "deleting")
	assert.Equal(t, "status <> failed", filter)
}
