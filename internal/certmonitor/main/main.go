package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/acs-fleet-manager/internal/certmonitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// resyncPeriod var is defualt resync period for informer
	var resyncPeriod = time.Second * 30
	// certificate monitor configuration
	config := &certmonitor.Config{
		// path to kubeconfig file
		Kubeconfig: filepath.Join(homedir.HomeDir(), ".kube", "config"),
		// list of monitors containing namespaces + secret configs
		Monitors: []certmonitor.MonitorConfig{
			{
				Namespace: certmonitor.SelectorConfig{
					Name: "namespace-three",
				},
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "secret-three-cert", // pragma: allowlist secret
				},
			},
			{
				Namespace: certmonitor.SelectorConfig{
					Name: "namespace-four",
				},
				Secret: certmonitor.SelectorConfig{
					Name: "secret-three-cert2",
				},
			},
			{
				Namespace: certmonitor.SelectorConfig{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"bar": "qux",
						},
					},
				},
				Secret: certmonitor.SelectorConfig{
					Name: "secret-labeled-1",
				},
			},
		},
	}
	if config.ResyncPeriod != nil {
		resyncPeriod = *config.ResyncPeriod
	}
	kubeconfigPath := config.Kubeconfig
	// build kubernetes client config
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("error building kubeconfig: %s", err)
		return
	}
	// create new kubernetes clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("error creating clientset: %s", err.Error())
		return
	}
	// validate cert monitor config
	if errs := certmonitor.ValidateConfig(*config); len(errs) != 0 {
		fmt.Printf("error validating config:\n")
		return
	}
	// create new informer factory, and informer for secrets
	informerFactory := informers.NewSharedInformerFactory(clientset, resyncPeriod)
	podInformer := informerFactory.Core().V1().Secrets().Informer()
	// lister for namespaces
	namespaceLister := informerFactory.Core().V1().Namespaces().Lister()
	//new cert monitor with (config, informerFactory, podInformer, namespaceLister)
	monitor, err := certmonitor.NewCertMonitor(config, informerFactory, podInformer, namespaceLister)
	if err != nil {
		fmt.Printf("Error creating certificate monitor: %v\n", err)
		os.Exit(1)
	}
	stopCh := make(chan struct{})
	defer close(stopCh)

	//start informer
	go monitor.StartInformer(stopCh)

	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Beg. to connect to port")
	http.ListenAndServe(":9091", nil)
}
