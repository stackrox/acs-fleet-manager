package util

import (
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIncrementCentralRevision(t *testing.T) {
	cases := map[string]struct {
		annotations      map[string]string
		expectedRevision string
		shouldFail       bool
	}{
		"empty revision should lead to revision == 1": {
			annotations:      map[string]string{},
			expectedRevision: "1",
		},
		"revision == 1 should increment to revision == 2": {
			annotations: map[string]string{
				RevisionAnnotationKey: "1",
			},
			expectedRevision: "2",
		},
		"non-integer string for revision should fail to increment": {
			annotations: map[string]string{
				RevisionAnnotationKey: "something",
			},
			shouldFail: true,
		},
		"nil annotations should lead to revision == 1": {
			annotations:      nil,
			expectedRevision: "1",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			central := &v1alpha1.Central{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: c.annotations,
				},
			}
			err := IncrementCentralRevision(central)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedRevision, central.GetAnnotations()[RevisionAnnotationKey])
			}
		})
	}
}

func TestIncrementCentralRevisionNilCentral(t *testing.T) {
	err := IncrementCentralRevision(nil)
	assert.Error(t, err)
}
