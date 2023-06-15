package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/exp/slices"
)

// Reconciler keeps necessary dependencies for the operator Reconciler
type Reconciler struct {
	operatorManager *ACSOperatorManager
}

// Reconcile takes a list of desired operator versions and makes sure that current cluster state has those operator versions
func (r *Reconciler) Reconcile(ctx context.Context, targetImages []string, crdTag string) ([]string, error) {
	currentImages, err := r.operatorManager.ListVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list operator versions: %w", err)
	}

	// collect operator images for installation
	var installImages []string
	for _, img := range targetImages {
		if !slices.Contains(currentImages, img) {
			installImages = append(installImages, img)
		}
	}
	if len(installImages) > 0 {
		glog.Infof("Installing operator versions: %s", strings.Join(installImages, ","))
		installOperatorImages := toOperatorImage(installImages, crdTag)
		err = r.operatorManager.InstallOrUpgrade(ctx, installOperatorImages)
		if err != nil {
			glog.Warningf("Failed installing operator versions: %v", err)
		}
	}

	// collect operator images for deletion
	var deleteImages []string
	for _, img := range currentImages {
		if !slices.Contains(targetImages, img) {
			deleteImages = append(deleteImages, img)
		}
	}
	if len(deleteImages) > 0 {
		glog.Infof("Deleting operator versions: %s", strings.Join(installImages, ","))
		deleteOperatorImages := toOperatorImage(deleteImages, crdTag)
		err = r.operatorManager.Delete(ctx, deleteOperatorImages)
		if err != nil {
			glog.Warningf("Failed deleting operator versions: %v", err)
		}
	}

	if len(installImages) == 0 && len(deleteImages) == 0 {
		return currentImages, nil
	}

	updatedImages, err := r.operatorManager.ListVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list operator versions: %w", err)
	}

	return updatedImages, nil
}

// NewOperatorReconciler creates a new Operator Reconciler instance
func NewOperatorReconciler(operatorManager *ACSOperatorManager) *Reconciler {
	return &Reconciler{operatorManager}
}
