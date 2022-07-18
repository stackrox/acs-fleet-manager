package db

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

func TestSQLTranslation(t *testing.T) {
	RegisterTestingT(t)

	disallowedFields := map[string]string{}
	filter, tables, err := ArgsToSearchFilter("id in ('123')", disallowedFields)
	Expect(err).ToNot(HaveOccurred())
	sql, values, err := filter.ToSql()
	Expect(err).ToNot(HaveOccurred())
	Expect(sql).To(Equal("id IN (?)"))
	Expect(values).To(ConsistOf("123"))
	Expect(tables).To(BeEmpty())
}

func TestSQLTranslationFailure(t *testing.T) {
	RegisterTestingT(t)

	disallowedFields := map[string]string{}
	_, _, err := ArgsToSearchFilter("garbage", disallowedFields)
	Expect(err).To(HaveOccurred())
	serviceErr := err.(*errors.ServiceError)
	Expect(serviceErr.Code).To(Equal(errors.ErrorBadRequest))
	Expect(serviceErr.Error()).To(Equal("RHACS-MGMT-21: Failed to parse search query: garbage"))
}

func TestDisallowedFields(t *testing.T) {
	RegisterTestingT(t)

	disallowedFields := map[string]string{
		"id": "id",
	}
	_, _, err := ArgsToSearchFilter("id in ('123')", disallowedFields)
	Expect(err).To(HaveOccurred())
	serviceErr := err.(*errors.ServiceError)
	Expect(serviceErr.Code).To(Equal(errors.ErrorBadRequest))
	Expect(serviceErr.Error()).To(Equal("RHACS-MGMT-21: id is not a valid field name"))
}

func TestTableNameInFields(t *testing.T) {
	RegisterTestingT(t)

	// it should succeed for valid search
	disallowedFields := map[string]string{}
	filter, tables, err := ArgsToSearchFilter("subscriptions.id in ('123') and subscription_labels.key = 'foo' and subscription_labels.value = 'bar'", disallowedFields)
	Expect(err).ToNot(HaveOccurred())
	sql, values, err := filter.ToSql()
	Expect(err).ToNot(HaveOccurred())
	Expect(sql).To(Equal("((subscriptions.id IN (?) AND subscription_labels.key = ?) AND subscription_labels.value = ?)"))
	Expect(values).To(ConsistOf("123", "foo", "bar"))
	Expect(tables).To(ConsistOf("subscriptions", "subscription_labels"))

	// it should fail if the field contains too many dots
	disallowedFields = map[string]string{}
	_, _, err = ArgsToSearchFilter("accounts.subscriptions.id", disallowedFields)
	Expect(err).To(HaveOccurred())
	serviceErr := err.(*errors.ServiceError)
	Expect(serviceErr.Code).To(Equal(errors.ErrorBadRequest))
	Expect(serviceErr.Error()).To(Equal("RHACS-MGMT-21: Failed to parse search query: accounts.subscriptions.id"))

	// it should fail for disallowed fields
	disallowedFields = map[string]string{
		"id": "id",
	}
	_, _, err = ArgsToSearchFilter("accounts.id in ('123')", disallowedFields)
	Expect(err).To(HaveOccurred())
	serviceErr = err.(*errors.ServiceError)
	Expect(serviceErr.Code).To(Equal(errors.ErrorBadRequest))
	Expect(serviceErr.Error()).To(Equal("RHACS-MGMT-21: accounts.id is not a valid field name"))
}
