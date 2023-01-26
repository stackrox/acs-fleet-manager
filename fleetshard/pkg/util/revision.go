package util

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
)

const (
	// RevisionAnnotationKey corresponds to the annotation key that holds the current revision number.
	RevisionAnnotationKey = "rhacs.redhat.com/revision"
)

// IncrementCentralRevision will increment the central's revision within its annotations.
// In case no revision annotation exists, the revision will be set to 1.
func IncrementCentralRevision(central *v1alpha1.Central) error {
	revisionString, ok := central.GetAnnotations()[RevisionAnnotationKey]
	// Annotation does not exist yet, create the initial revision of 1.
	if !ok {
		central.GetAnnotations()[RevisionAnnotationKey] = "1"
		return nil
	}

	revision, err := strconv.Atoi(revisionString)
	if err != nil {
		return errors.Wrapf(err, "failed to increment central revision %s", central.GetName())
	}
	revision++
	central.Annotations[RevisionAnnotationKey] = fmt.Sprintf("%d", revision)
	return nil
}
