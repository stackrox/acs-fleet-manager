package certmonitor

import "C"
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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

var (
	certificatesExpiry = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "acscs_certmonitor_certificate_expiration_timestamp",
		Help: "Expiry of certifications",
	},
		[]string{"namespace", "secret", "data_key"},
	)
)

const (
	labelSecretName      = "name"
	labelSecretNamespace = "namespace"
	labelDataKey         = "data_key"
)

type SelectorConfig struct {
	Name          string                `json:"name"`
	LabelSelector *metav1.LabelSelector `json:"labelSelector"`
}

type MonitorConfig struct {
	Namespace SelectorConfig `json:"namespace"`
	Secret    SelectorConfig `json:"secret"`
}

type Config struct {
	Monitors     []MonitorConfig `json:"monitors"`
	Kubeconfig   string          `json:"kubeconfig"`
	ResyncPeriod *time.Duration  `json:"resyncPeriod"`
}

type NamespaceGetter interface {
	Get(name string) (*corev1.Namespace, error)
}

type certMonitor struct {
	clientset       *kubernetes.Clientset
	informedfactory informers.SharedInformerFactory
	podInformer     cache.SharedIndexInformer
	config          *Config
	namespaceLister NamespaceGetter
}

func init() {
	prometheus.MustRegister(certificatesExpiry)
}

func (c *certMonitor) StartInformer(stopCh <-chan struct{}) {
	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleSecretCreation,
		UpdateFunc: c.handleSecretUpdate,
		DeleteFunc: c.handleSecretDeletion,
	},
	)
	c.informedfactory.Start(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced) {
		fmt.Errorf("timed out waiting for caches to sync")
		return
	}
	fmt.Println("Synced and ready:")

}

func NewCertMonitor(config *Config, informedFactory informers.SharedInformerFactory, podInformer cache.SharedIndexInformer, namespaceLister v1.NamespaceLister) (*certMonitor, error) {
	monitor := &certMonitor{
		informedfactory: informedFactory,
		podInformer:     podInformer,
		config:          config,
		namespaceLister: namespaceLister,
	}
	return monitor, nil
}

func (c *certMonitor) findCertsFromInformer(new interface{}) {
	secret, ok := new.(*corev1.Secret)
	if !ok {
		fmt.Printf("Secret Error, got: %v\n", new)
		return
	}

	for _, monitor := range c.config.Monitors {
		if c.secretMatches(secret, monitor) {
			c.findCertsFromSecret(secret)
			break
		}
	}

}

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

func (c *certMonitor) findCertsFromSecret(secret *corev1.Secret) {
	for dataKey, dataCert := range secret.Data {
		certConv, err := base64.StdEncoding.DecodeString(string(dataCert))
		if err != nil {
			fmt.Printf("failed to decode base64 data: %v", err)
			continue
		}

		pparse, _ := pem.Decode(certConv)
		if pparse == nil {
			fmt.Printf("failed to decode pem data on secret %s/%s/%s\n", secret.Namespace, secret.Name, dataKey)
			continue
		}

		certss, err := x509.ParseCertificate(pparse.Bytes)
		if err != nil {
			fmt.Printf("failed to decode certificate on secret %s/%s/%s\n", secret.Namespace, secret.Name, dataKey)
			continue
		}

		expiryTime := float64(certss.NotAfter.Unix())
		certificatesExpiry.WithLabelValues(secret.Namespace, secret.Name, dataKey).Set(expiryTime)
		fmt.Println("Certificates Expiry Date found:")
		fmt.Println(certss.NotAfter)
	}
}

//

func (c *certMonitor) handleSecretCreation(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		fmt.Printf("unexpected object: %v", obj)
		return
	}
	fmt.Printf("Handling Creation: %s,%s\n", secret.Namespace, secret.Name)
	c.findCertsFromSecret(secret)
}

func (c *certMonitor) handleSecretUpdate(oldObj, newObj interface{}) {
	oldsecret, ok := oldObj.(*corev1.Secret)
	if !ok {
		fmt.Printf("unexpected object: %v", oldObj)
		return
	}

	newsecret, ok := newObj.(*corev1.Secret)
	if !ok {
		fmt.Printf("unexpected object: %v", newObj)
		return
	}

	if oldObj != nil {
		for oldKey := range oldsecret.Data {
			if _, ok := newsecret.Data[oldKey]; !ok {
				certificatesExpiry.DeletePartialMatch(prometheus.Labels{
					labelSecretName:      newsecret.Name,
					labelSecretNamespace: newsecret.Namespace,
					labelDataKey:         oldKey,
				})
			}
		}
	}

	fmt.Printf("Handling update: %s,%s\n", newsecret.Namespace, newsecret.Name)
	c.findCertsFromSecret(newsecret)
}

func (c *certMonitor) handleSecretDeletion(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		fmt.Printf("unexpected object: %v", obj)
		return
	}

	for dataKey := range secret.Data {
		certificatesExpiry.DeletePartialMatch(prometheus.Labels{
			labelSecretName:      secret.Name,
			labelSecretNamespace: secret.Namespace,
			labelDataKey:         dataKey,
		})
	}
	certificatesExpiry.DeletePartialMatch(prometheus.Labels{
		labelSecretName:      secret.Name,
		labelSecretNamespace: secret.Namespace,
	})

	fmt.Printf("Handling Delete: %s,%s\n", secret.Namespace, secret.Name)
}

/*


func ValidateConfig(config Config) (errs field.ErrorList) {
	errs = append(errs, validateMonitors(field.NewPath("monitors"), config.Monitors)..)
	return errs
}

func validateMonitors(path *field.Path, monitors []Monitor) (errs field.ErrorList) {
	for i, monitor := range monitors {
		errs = append(errs, validateMonitor(path.Index(i), monitor)...)
	}
	return errs
}

func validateMonitor(path *field.Path, monitor Monitor) (errs field.ErrorList) {
	errs = append(errs, path.Child("namespace"), validateSelectorConfig(monitor.Namespace)...)
	errs = append(errs, path.Child("secret"), validateSelectorConfig(monitor.Secret)...)
	return errs
}

func validateSelectorConfig(path *field.Path, selectorConfig SelectorConfig) (errs field.ErrorList) {
	if len(selectorConfig.Name) != 0 && len(selectorConfig.LabelSelector) != 0 {
		errs = append(errs, field.Invalid(field, selectorConfig, "cannot specify both name and label selector)
	}
	return errs
}

*/
