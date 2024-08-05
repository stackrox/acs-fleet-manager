package certmonitor

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/informers"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

// define Prometheus metrics
var (
	certificatesExpiry = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "acscs_certmonitor_certificate_expiration_timestamp",
		Help: "Expiry of certifications",
	},
		[]string{"namespace", "secret", "data_key"},
	)
)

// const variables for deletion selection
const (
	labelSecretName      = "name"      // pragma: allowlist secret
	labelSecretNamespace = "namespace" // pragma: allowlist secret
	labelDataKey         = "data_key"
)

// SelectorConfig struct for namespace or secret selection based on labels/name
type SelectorConfig struct {
	Name          string                `json:"name"`
	LabelSelector *metav1.LabelSelector `json:"labelSelector"`
}

// MonitorConfig struct for monitoring specific namespaces + secrets
type MonitorConfig struct {
	Namespace SelectorConfig `json:"namespace"`
	Secret    SelectorConfig `json:"secret"`
}

// Config struct, overall configuration for the certificate monitoring
type Config struct {
	Monitors     []MonitorConfig `json:"monitors"`
	Kubeconfig   string          `json:"kubeconfig"`
	ResyncPeriod *time.Duration  `json:"resyncPeriod"`
}

// NamespaceGetter interface for retrieving namespaces on name
type NamespaceGetter interface {
	Get(name string) (*corev1.Namespace, error)
}

// certMonitor struct is main struct for certificate monitoring
type certMonitor struct {
	informerfactory informers.SharedInformerFactory
	podInformer     cache.SharedIndexInformer
	config          *Config
	namespaceLister NamespaceGetter
}

// init registers certifcatesExpiry with prometheus
func init() {
	prometheus.MustRegister(certificatesExpiry)
}

// StartInformer func starts to informer to monitor secrets + event handlers
func (c *certMonitor) StartInformer(stopCh <-chan struct{}) error {
	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleSecretCreation,
		UpdateFunc: c.handleSecretUpdate,
		DeleteFunc: c.handleSecretDeletion,
	},
	)
	c.informerfactory.Start(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}
	return nil
}

// NewCertMonitor func creates new instance of certMonitor
func NewCertMonitor(config *Config, informerFactory informers.SharedInformerFactory, podInformer cache.SharedIndexInformer, namespaceLister v1.NamespaceLister) (*certMonitor, error) {
	monitor := &certMonitor{
		informerfactory: informerFactory,
		podInformer:     podInformer,
		config:          config,
		namespaceLister: namespaceLister,
	}
	return monitor, nil
}

// objectMatchesSelector func checks if object matches given label selector
func objectMatchesSelector(obj runtime.Object, selector *metav1.LabelSelector) bool {
	if selector == nil {
		return true
	}
	labelselector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false
	}

	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return false
	}

	return labelselector.Matches(labels.Set(metaObj.GetLabels()))

}

// secretMatches func checks if secret matches in the monitor config
func (c *certMonitor) secretMatches(s *corev1.Secret, monitor MonitorConfig) bool {
	if s == nil {
		return false
	}
	if len(monitor.Secret.Name) > 0 && s.Name != monitor.Secret.Name {
		return false
	}
	if len(monitor.Namespace.Name) > 0 && s.Namespace != monitor.Namespace.Name {
		return false
	}
	if monitor.Secret.LabelSelector != nil && !objectMatchesSelector(s, monitor.Secret.LabelSelector) {
		return false
	}

	if monitor.Namespace.LabelSelector != nil {
		ns, err := c.namespaceLister.Get(s.Namespace)
		if err != nil {
			return false
		}
		if !objectMatchesSelector(ns, monitor.Secret.LabelSelector) {
			return false
		}
	}
	return true
}

// findCertsFromSecret func extracts, decodes, parses certifications from a secret, and displays in Prometheus
func (c *certMonitor) findCertsFromSecret(secret *corev1.Secret) {
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
			continue
		}

		expiryTime := float64(certss.NotAfter.Unix())
		certificatesExpiry.WithLabelValues(secret.Namespace, secret.Name, dataKey).Set(expiryTime)
	}
}

// handleScretCreation func event handles new secret creations
func (c *certMonitor) handleSecretCreation(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}
	fmt.Printf("Handling Creation: %s,%s\n", secret.Namespace, secret.Name)
	c.findCertsFromSecret(secret)
}

// handleSecretUpdate func event handles secret updates
func (c *certMonitor) handleSecretUpdate(oldObj, newObj interface{}) {
	oldsecret, ok := oldObj.(*corev1.Secret)
	if !ok {
		return
	}

	newsecret, ok := newObj.(*corev1.Secret)
	if !ok {
		return
	}

	if newObj == nil || oldObj == nil {
		return
	}
	for oldKey := range oldsecret.Data {
		if _, ok := newsecret.Data[oldKey]; !ok {
			certificatesExpiry.DeletePartialMatch(prometheus.Labels{
				labelSecretName:      newsecret.Name,      // pragma: allowlist secret
				labelSecretNamespace: newsecret.Namespace, // pragma: allowlist secret
				labelDataKey:         oldKey,
			})
		}
	}
	c.findCertsFromSecret(newsecret)
}

// handleSecretDeletion func event handles deletion of secrets
func (c *certMonitor) handleSecretDeletion(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return
	}

	for dataKey := range secret.Data {
		certificatesExpiry.DeletePartialMatch(prometheus.Labels{
			labelSecretName:      secret.Name,      // pragma: allowlist secret
			labelSecretNamespace: secret.Namespace, // pragma: allowlist secret
			labelDataKey:         dataKey,
		})
	}
	certificatesExpiry.DeletePartialMatch(prometheus.Labels{
		labelSecretName:      secret.Name,      // pragma: allowlist secret
		labelSecretNamespace: secret.Namespace, // pragma: allowlist secret
	})
}

// ValidateConfig func checks the validity of given config,  in 'Monitor'
func ValidateConfig(config Config) (errs field.ErrorList) {
	errs = append(errs, validateMonitors(field.NewPath("monitors"), config.Monitors)...)
	return errs
}

// validateMonitors func validates list of Monitor
func validateMonitors(path *field.Path, monitors []MonitorConfig) (errs field.ErrorList) {
	for i, monitor := range monitors {
		errs = append(errs, validateMonitor(path.Index(i), monitor)...)
	}
	return errs
}

// validateMonitor func validates single Monitor obj, including: 'Namespace' and 'Secret'
func validateMonitor(path *field.Path, monitor MonitorConfig) (errs field.ErrorList) {
	errs = append(errs, validateSelectorConfig(path.Child("namespace"), monitor.Namespace)...)
	errs = append(errs, validateSelectorConfig(path.Child("secret"), monitor.Secret)...)
	return errs
}

// validateSelectorConfig func validates selectorConfig obj
func validateSelectorConfig(path *field.Path, selectorConfig SelectorConfig) (errs field.ErrorList) {
	if len(selectorConfig.Name) != 0 && selectorConfig.LabelSelector != nil {
		errs = append(errs, field.Invalid(path, selectorConfig, "cannot specify both name and label selector"))
	}
	return errs
}
