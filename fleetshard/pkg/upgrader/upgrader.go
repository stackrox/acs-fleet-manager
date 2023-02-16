package upgrader

/**
 - How is the upgrader notified?
 - How does the upgrader report the upgrade status?
 	- Expose a Metric?
 - What happens if the reconciler schedules a Central during an upgrade?
   - Ignore pause-reconcile, spin up instance

1)
 - Add annotation to each Central canary group
 - Reconciler

 - Process: List all Central instances

*/

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"os"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	pauseReconcileAnnotation = "stackrox.io/pause-reconcile"
)

var (
	availableGroups = []string{"A", "B"}
)

type CanaryUpgrader struct {
	k8sClient ctrlClient.Client
}

// NewCanaryUpgrader ...
func NewCanaryUpgrader(client ctrlClient.Client) CanaryUpgrader {
	return CanaryUpgrader{
		k8sClient: client,
	}
}

func (c *CanaryUpgrader) Manager() error {
	// Phase 1: set pause reconcile
	// Phase 2: upgrade operator
	// Phase 3: upgrade Central group

	// how to recover upgrade process after interrupt?
	// how to rollback on failure?

	ctx := context.TODO()
	err := c.SetPauseReconcileAnnotation(ctx)
	if err != nil {
		// TODO: retry and rollback Central instances
		return err
	}

	// TODO: Trigger operator upgrade (OLM, Helm etc)
	// TODO: Check operator health
	glog.Infof("Upgrading operator...")
	time.Sleep(5 * time.Second)
	glog.Infof("Operator upgraded...")
	os.Exit(0)

	for _, group := range availableGroups {
		glog.Infof("Upgrade Central Group %s", group)
		// TODO: Filter list by group label
		list := &platform.CentralList{}
		err := c.k8sClient.List(ctx, list)
		if err != nil {
			return fmt.Errorf("listing central: %w", err)
		}

		// TODO: wait until group is finished?
		// TODO: Do it async?
		err = c.StartUpgrade(ctx, &platform.Central{})
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO: Possible abstraction to work on generic k8s metadata objects
// TODO: Only use k8s objects, define ready function when a deployment is considered healthy
func (c *CanaryUpgrader) SetPauseReconcileAnnotation(ctx context.Context) error {
	list := &platform.CentralList{}
	err := c.k8sClient.List(ctx, list)
	if err != nil {
		return fmt.Errorf("listing central: %w", err)
	}

	for _, central := range list.Items {
		central.Annotations[pauseReconcileAnnotation] = "true"
		err := c.k8sClient.Update(ctx, &central)
		if err != nil {
			return fmt.Errorf("setting pause reconcile annotation: %w", err)
		}
	}

	return nil
}

func (c *CanaryUpgrader) StartUpgrade(ctx context.Context, central *platform.Central) error {
	c.SetPauseReconcileAnnotation(ctx)

	// Filter for Centrals by Group label to execute ordered upgrades
	if _, ok := central.Annotations[pauseReconcileAnnotation]; !ok {
		return fmt.Errorf("instance is not upgrade candidate: %s", central.GetName())
	}

	delete(central.Annotations, pauseReconcileAnnotation)
	err := c.k8sClient.Update(ctx, central)
	if err != nil {
		return fmt.Errorf("setting pause reconcile annotation: %w", err)
	}

	return nil
}

func (c *CanaryUpgrader) DeploymentInformer(client kubernetes.Interface) error {
	factory := informers.NewSharedInformerFactory(client, 1*time.Hour)
	informer := factory.Apps().V1().Deployments().Informer()
	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()

	//TODO: check - ready replicas are not always send as update, last update is missing
	//TODO: check maintaining map of deployments, or multiplexer to send deployment updates to upgrade go-routines
	// and dispatching them
	// TODO: relation between deployment and Central CR necessary to complete upgrade
	// TODO: Check Scanner health
	glog.Infof("registering informer")
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			depl := obj.(*v1.Deployment)
			glog.Infof("Add received for object: %s/%s, Ready Replicas: %d/%d", depl.GetNamespace(), depl.GetName(), depl.Status.AvailableReplicas, depl.Status.Replicas)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			depl := newObj.(*v1.Deployment)
			glog.Infof("Update received for object: %s/%s, Ready Replicas: %d/%d", depl.GetNamespace(), depl.GetName(), depl.Status.ReadyReplicas, depl.Status.Replicas)
			if depl.Status.ReadyReplicas == depl.Status.Replicas {
				glog.Infof("Upgraded %s/%s", depl.GetNamespace(), depl.GetName())
			}
		},
	})
	if err != nil {
		return fmt.Errorf("registering informer", err)
	}
	go informer.Run(stopper)

	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return nil
	}

	<-stopper
	return nil
}
