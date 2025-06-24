// Package main entrypoint for the fleetshard operator
package main

import (
	"crypto/tls"
	"flag"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-logr/glogr"
	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/fleetshard-operator/api/v1alpha1"
	"github.com/stackrox/acs-fleet-manager/fleetshard-operator/pkg/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
)

func main() {
	defer glog.Flush()

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var sourceNamespace string
	var enableHTTP2 = false

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableHTTP2, "enable-http2", enableHTTP2, "If HTTP/2 should be enabled for the metrics and webhook servers.")
	flag.StringVar(&sourceNamespace, "source-namespace", "rhacs", "namespace where the repository secret is located")
	flag.Parse()
	ctrl.SetLogger(glogr.New())

	disableHTTP2 := func(c *tls.Config) {
		if enableHTTP2 {
			return
		}
		c.NextProtos = []string{"http/1.1"}
	}
	webhookServerOptions := webhook.Options{
		TLSOpts: []func(config *tls.Config){disableHTTP2},
		Port:    9443,
	}
	webhookServer := webhook.NewServer(webhookServerOptions)

	metricsServerOptions := metricsserver.Options{
		BindAddress: metricsAddr,
		TLSOpts:     []func(*tls.Config){disableHTTP2},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "16e7a03e.cloud.stackrox.io",
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				sourceNamespace:                     {},
				controllers.ArgoCdNamespace:         {},
				controllers.GitopsOperatorNamespace: {},
			},
		},
	})
	if err != nil {
		glog.Fatalf("unable to start manager: %v", err)
	}

	reconciler := &controllers.GitopsInstallationReconciler{
		Client:          mgr.GetClient(),
		SourceNamespace: sourceNamespace,
	}
	if err = reconciler.SetupWithManager(mgr); err != nil {
		glog.Fatalf("unable to create GitopsInstallation controller")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		glog.Fatalf("unable to set up health check: %v", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		glog.Fatalf("unable to set up ready check: %v", err)
	}
	glog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		glog.Fatalf("problem running manager: %v", err)
	}
}

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Infof("Unable to set logtostderr to true")
	}
	utilRuntime.Must(clientgoscheme.AddToScheme(scheme))
	utilRuntime.Must(v1alpha1.AddToScheme(scheme))
	utilRuntime.Must(argoCd.AddToScheme(scheme))
	utilRuntime.Must(operatorsv1alpha1.AddToScheme(scheme))
	utilRuntime.Must(operatorsv1.AddToScheme(scheme))
	utilRuntime.Must(configv1.AddToScheme(scheme))
}
