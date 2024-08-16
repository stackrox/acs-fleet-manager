// Package main ...
package main

import (
	"context"
	"flag"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/reconciler"
	"github.com/stackrox/acs-fleet-manager/internal/certmonitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"os"
	"os/signal"
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

	tenantNamespaceSelector := certmonitor.SelectorConfig{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      reconciler.TenantIDLabelKey,
					Operator: metav1.LabelSelectorOpExists,
				},
			},
		},
	}
	certmonitorConfig := &certmonitor.Config{
		Monitors: []certmonitor.MonitorConfig{
			{
				Namespace: tenantNamespaceSelector,
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "scanner-tls",
				},
			},
			{
				Namespace: tenantNamespaceSelector,
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "central-tls",
				},
			},

			{
				Namespace: tenantNamespaceSelector,
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "scanner-db-tls",
				},
			},

			{
				Namespace: tenantNamespaceSelector,
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "scanner-v4-db-tls",
				},
			},

			{
				Namespace: tenantNamespaceSelector,
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "scanner-v4-indexer-tls",
				},
			},
			{
				Namespace: tenantNamespaceSelector,
				Secret: certmonitor.SelectorConfig{ // pragma: allowlist secret
					Name: "scanner-v4-matcher-tls",
				},
			},
		},
	}

	if errs := certmonitor.ValidateConfig(*certmonitorConfig); len(errs) > 0 {
		glog.Fatalf("certmonitor validation error: %v", errs)
	}

	k8sInterface := k8s.CreateInterfaceOrDie()
	informedFactory := informers.NewSharedInformerFactory(k8sInterface, time.Minute)
	secretInformer := informedFactory.Core().V1().Secrets().Informer()
	namespaceLister := informedFactory.Core().V1().Namespaces().Lister()

	monitor := certmonitor.NewCertMonitor(certmonitorConfig, informedFactory, secretInformer, namespaceLister)

	if err := monitor.Start(); err != nil {
		glog.Fatalf("Error starting certmonitor: %v", err)
	}

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

	if err := monitor.Stop(); err != nil {
		glog.Errorf("Error stoping certmonitor: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Fleetshard-sync application has been stopped")
}
