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
	var resyncPeriod = time.Second * 30
	config := &certmonitor.Config{

		Kubeconfig: filepath.Join(homedir.HomeDir(), ".kube", "config"),
		Monitors: []certmonitor.MonitorConfig{
			{
				Namespace: certmonitor.SelectorConfig{
					Name: "namespace-three",
				},
				Secret: certmonitor.SelectorConfig{
					Name: "secret-three-cert",
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
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Errorf("error building kubeconfig: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Errorf("error creating clientset: %s", err.Error())
	}

	if errs := certmonitor.ValidateConfig(*config); len(errs) != 0 {
		fmt.Errorf("error validating config:\n")

	}

	informerFactory := informers.NewSharedInformerFactory(clientset, resyncPeriod)
	podInformer := informerFactory.Core().V1().Secrets().Informer()
	namespaceLister := informerFactory.Core().V1().Namespaces().Lister()

	monitor, err := certmonitor.NewCertMonitor(config, informerFactory, podInformer, namespaceLister)
	if err != nil {
		fmt.Printf("Error creating certificate monitor: %v\n", err)
		os.Exit(1)
	}
	stopCh := make(chan struct{})
	defer close(stopCh)

	go monitor.StartInformer(stopCh)

	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Beg. to connect to port")
	http.ListenAndServe(":9091", nil)
}
