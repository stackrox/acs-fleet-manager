// Package main ...
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/certmonitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/runtime"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	config, err := config.GetConfig()
	if err != nil {
		glog.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.StartupTimeout)
	defer cancel()
	glog.Infof("Starting application, timeout=%s", config.StartupTimeout)
	glog.Infof("FleetManagerEndpoint: %s", config.FleetManagerEndpoint)
	glog.Infof("ClusterID: %s", config.ClusterID)
	glog.Infof("RuntimePollPeriod: %s", config.RuntimePollPeriod.String())
	glog.Infof("AuthType: %s", config.AuthType)
	glog.Infof("ManagedDB.Enabled: %t", config.ManagedDB.Enabled)
	glog.Infof("ManagedDB.SecurityGroup: %s", config.ManagedDB.SecurityGroup)
	glog.Infof("ManagedDB.SubnetGroup: %s", config.ManagedDB.SubnetGroup)
	if len(config.TenantImagePullSecret) > 0 {
		glog.Infof("Image pull secret configured, will be injected into tenant namespaces.")
	}
	glog.Info("Creating k8s client...")
	k8sClient := k8s.CreateClientOrDie()
	ctrl.SetLogger(logger.NewKubeAPILogger())
	glog.Info("Creating runtime...")
	runtime, err := runtime.NewRuntime(ctx, config, k8sClient)
	if err != nil {
		glog.Fatal(err)
	}

	go func() {
		err := runtime.Start()
		if err != nil {
			glog.Fatal(err)
		}
	}()

	glog.Info("Creating certMonitor")

	certmonitorConfig := &certmonitor.Config{

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
	kubeconfigPath := certmonitorConfig.Kubeconfig
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Errorf("error building kubeconfig: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Errorf("error creating clientset: %s", err.Error())
	}

	/*
		if errs := certmonitor.ValidateConfig(*certmonitorConfig); len(errs) != 0 {
			fmt.Errorf("error validating config:\n")
			for _, err := range errs {
				fmt.Printf("%s\n", err)
			}
		}

	*/

	informedFactory := informers.NewSharedInformerFactory(clientset, time.Minute)
	podInformer := informedFactory.Core().V1().Secrets().Informer()
	namespaceLister := informedFactory.Core().V1().Namespaces().Lister()

	monitor, err := certmonitor.NewCertMonitor(certmonitorConfig, informedFactory, podInformer, namespaceLister)
	if err != nil {
		fmt.Printf("Error creating certificate monitor: %v\n", err)
		os.Exit(1)
	}
	stopCh := make(chan struct{})
	go monitor.StartInformer(stopCh)

	glog.Info("Creating metrics server...")
	metricServer := fleetshardmetrics.NewMetricsServer(config.MetricsAddress)
	go func() {
		if err := metricServer.ListenAndServe(); err != nil {
			glog.Errorf("serving metrics server: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	notifySignals := []os.Signal{os.Interrupt, unix.SIGTERM}
	signal.Notify(sigs, notifySignals...)

	glog.Infof("Application started. Will shut down gracefully on %s.", notifySignals)
	sig := <-sigs
	runtime.Stop()
	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Fleetshard-sync application has been stopped")
}
