package certmonitor

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"time"
)

// SelectorConfig represents a configuration to select a namespace or a secret by name or labelSelector. Only one of Name or LabelSelector can be specified.
type SelectorConfig struct {
	Name          string                `json:"name"`
	LabelSelector *metav1.LabelSelector `json:"labelSelector"`
}

// MonitorConfig represents a configuration for observing certificates contained in kubernetes secrets
type MonitorConfig struct {
	Namespace SelectorConfig `json:"namespace"`
	Secret    SelectorConfig `json:"secret"`
}

// Config represents the certificate monitor configuration
type Config struct {
	Monitors     []MonitorConfig `json:"monitors"`
	ResyncPeriod *time.Duration  `json:"resyncPeriod"`
}

// NamespaceGetter interface for retrieving namespaces on name. This interface is a subset of `v1.NamespaceLister`
type NamespaceGetter interface {
	Get(name string) (*corev1.Namespace, error)
}

// certMonitor is the Certificate Monitor. It watches Kubernetes secrets containing certificates, and populates prometheus metrics with the expiration time of those certificates.
type certMonitor struct {
	informerfactory informers.SharedInformerFactory
	secretInformer  cache.SharedIndexInformer
	config          *Config
	namespaceGetter NamespaceGetter
	metrics         *fleetshardmetrics.Metrics
}

// Start the certificate monitor
func (c *certMonitor) Start(stopCh <-chan struct{}) error {
	c.secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleSecretCreation,
		UpdateFunc: c.handleSecretUpdate,
		DeleteFunc: c.handleSecretDeletion,
	})
	c.informerfactory.Start(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.secretInformer.HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}
	return nil
}

// NewCertMonitor creates new instance of certMonitor
func NewCertMonitor(config *Config, informerFactory informers.SharedInformerFactory, secretInformer cache.SharedIndexInformer, namespaceGetter NamespaceGetter) *certMonitor {
	return &certMonitor{
		informerfactory: informerFactory,
		secretInformer:  secretInformer, // pragma: allowlist secret
		config:          config,
		namespaceGetter: namespaceGetter,
		metrics:         fleetshardmetrics.MetricsInstance(),
	}
}

// processSecret extracts, decodes, parses certificates from a secret, and populates prometheus metrics
func (c *certMonitor) processSecret(secret *corev1.Secret) {
	for dataKey, dataCert := range secret.Data {
		certConv, err := base64.StdEncoding.DecodeString(string(dataCert))
		if err != nil {
			continue
		}

		pparse, _ := pem.Decode(certConv)
		if pparse == nil {
			continue
		}

		certss, err := x509.ParseCertificate(pparse.Bytes)
		if err != nil {
			glog.Errorf("Failed to parse certificate %s: %v", dataKey, err)
		}
		expiryTime := float64(certss.NotAfter.Unix())
		c.metrics.SetCertKeyExpiryMetric(secret.Namespace, secret.Name, dataKey, expiryTime)
	}
}

// handleSecretCreation handles secret creation events
func (c *certMonitor) handleSecretCreation(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}
	c.processSecret(secret)
}

// handleSecretUpdate handles secret updates
func (c *certMonitor) handleSecretUpdate(oldObj, newObj interface{}) {
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
	c.processSecret(newSecret)
}

// handleSecretDeletion handles deletion of secrets
func (c *certMonitor) handleSecretDeletion(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}
	c.metrics.DeleteCertMetric(secret.Namespace, secret.Name)
}

// ValidateConfig checks the validity of Config
func ValidateConfig(config Config) (errs field.ErrorList) {
	errs = append(errs, validateMonitors(field.NewPath("monitors"), config.Monitors)...)
	return errs
}

// validateMonitors validates list of Monitor
func validateMonitors(path *field.Path, monitors []MonitorConfig) (errs field.ErrorList) {
	for i, monitor := range monitors {
		errs = append(errs, validateMonitor(path.Index(i), monitor)...)
	}
	return errs
}

// validateMonitor validates a Monitor
func validateMonitor(path *field.Path, monitor MonitorConfig) (errs field.ErrorList) {
	errs = append(errs, validateSelectorConfig(path.Child("namespace"), monitor.Namespace)...)
	errs = append(errs, validateSelectorConfig(path.Child("secret"), monitor.Secret)...)
	return errs
}

// validateSelectorConfig validates a SelectorConfig
func validateSelectorConfig(path *field.Path, selectorConfig SelectorConfig) (errs field.ErrorList) {
	if len(selectorConfig.Name) != 0 && selectorConfig.LabelSelector != nil {
		errs = append(errs, field.Invalid(path, selectorConfig, "cannot specify both name and label selector"))
	}
	return errs
}
