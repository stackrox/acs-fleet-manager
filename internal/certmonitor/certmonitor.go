package certmonitor

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	tlsSecretLabel = "rhacs.redhat.com/tls" // pragma: allowlist secret
	syncPeriod     = 30 * time.Minute
)

// CertMonitor is the Certificate Monitor. It watches Kubernetes secrets with label rhacs.redhat.com/tls=true
// and populates prometheus metrics with the expiration time of those certificates.
type CertMonitor struct {
	cache   ctrlcache.Cache
	metrics *fleetshardmetrics.Metrics
	cancel  context.CancelFunc
}

// Start the certificate monitor
func (c *CertMonitor) Start() error {
	if c.cancel != nil {
		return errors.New("already started")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	// Add event handler for secret events
	informer, err := c.cache.GetInformer(ctx, &corev1.Secret{})
	if err != nil {
		return fmt.Errorf("failed to get secret informer: %w", err)
	}

	_, err = informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleSecretCreation,
		UpdateFunc: c.handleSecretUpdate,
		DeleteFunc: c.handleSecretDeletion,
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start the cache
	go func() {
		if err := c.cache.Start(ctx); err != nil {
			glog.Errorf("certmonitor cache error: %v", err)
		}
	}()

	// Wait for cache to sync
	if !c.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("timed out waiting for cache to sync")
	}

	glog.Info("Certificate monitor started successfully")
	return nil
}

func (c *CertMonitor) Stop() error {
	if c.cancel == nil {
		return errors.New("not started")
	}
	c.cancel()
	c.cancel = nil
	return nil
}

// NewCertMonitor creates new instance of CertMonitor
// The cache must be configured to only watch tenant tls secrets
func NewCertMonitor(restConfig *rest.Config) *CertMonitor {
	syncPeriod := syncPeriod
	cache, err := ctrlcache.New(restConfig, ctrlcache.Options{
		ByObject: map[client.Object]ctrlcache.ByObject{
			&corev1.Secret{}: {
				Label: labels.SelectorFromSet(labels.Set{
					tlsSecretLabel: "true",
				}),
			},
		},
		DefaultLabelSelector: labels.Nothing(), // Don't cache any other resources
		SyncPeriod:           &syncPeriod,      // Reduce sync frequency to 30 minutes
	})
	if err != nil {
		glog.Fatalf("Failed to create certmonitor cache: %v", err)
	}
	return &CertMonitor{
		cache:   cache,
		metrics: fleetshardmetrics.MetricsInstance(),
	}
}

// processSecret extracts, decodes, parses certificates from a secret, and populates prometheus metrics
func (c *CertMonitor) processSecret(secret *corev1.Secret) {
	for dataKey, dataCert := range secret.Data {

		pparse, _ := pem.Decode(dataCert)
		if pparse == nil {
			continue
		}

		certss, err := x509.ParseCertificate(pparse.Bytes)
		if err != nil {
			continue
		}

		expiryTime := float64(certss.NotAfter.Unix())
		c.metrics.SetCertKeyExpiryMetric(secret.Namespace, secret.Name, dataKey, expiryTime)
	}
}

func (c *CertMonitor) handleSecretCreation(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}
	c.processSecret(secret)
}

func (c *CertMonitor) handleSecretUpdate(oldObj, newObj interface{}) {
	oldSecret, ok := oldObj.(*corev1.Secret)
	if !ok {
		return
	}

	newSecret, ok := newObj.(*corev1.Secret)
	if !ok {
		return
	}

	if newObj == nil || oldObj == nil {
		return
	}

	for oldKey := range oldSecret.Data {
		if _, ok := newSecret.Data[oldKey]; !ok {
			// secret has been updated, and oldKey does not exist in the new secret - so we delete the metric
			c.metrics.DeleteKeyCertMetric(newSecret.Namespace, newSecret.Name, oldKey)
		}
	}

	// Process the updated secret
	c.processSecret(newSecret)
}

func (c *CertMonitor) handleSecretDeletion(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}
	c.metrics.DeleteCertMetric(secret.Namespace, secret.Name)
}
