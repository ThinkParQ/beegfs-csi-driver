/*
Copyright 2021 NetApp, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"github.com/netapp/beegfs-csi-driver/operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(beegfsv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	// Adapted from https://github.com/kubernetes-sigs/kubebuilder/discussions/3907.
	//
	// If the enable-http2 flag is false (the default), http/2 should be disabled due to its
	// vulnerabilities. More specifically, disabling http/2 will prevent from being vulnerable to
	// the HTTP/2 Stream Cancellation and Rapid Reset CVEs. For more information see:
	//  - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	//  - https://github.com/advisories/GHSA-4374-p667-p6c8
	var enableHTTP2 bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "Enable HTTP/2 for the metrics server.")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Watch all namespaces, or only the namespace provided by the Downward API:
	var defaultNamespaces map[string]cache.Config
	if driverNamespace := os.Getenv("BEEGFS_CSI_DRIVER_NAMESPACE"); driverNamespace != "" {
		defaultNamespaces = map[string]cache.Config{
			driverNamespace: {},
		}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			setupLog.Info("disabling http/2")
			c.NextProtos = []string{"http/1.1"}
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		// The old Namespace field was deprecated in:
		// https://github.com/kubernetes-sigs/controller-runtime/commit/0377e0f7a41bd9d88d43811192309b193c9cd793#diff-d500fbd6a2aa620607ca5e2a7c3ac4f1a4c82309d1a549561e92abfcb18f2f0e.
		// More details are in the commit that added the DefaultNamespaces field to cache.Options:
		// https://github.com/kubernetes-sigs/controller-runtime/commit/3e35cab1ea29145d8699259b079633a2b8cfc116#diff-964e351ee2375d359c78d69e514c4edc42577219761c4475f391ed2daf715e51.
		Cache: cache.Options{
			DefaultNamespaces: defaultNamespaces,
		},
		// The old MetricsBindAddress field was deprecated in:
		// https://github.com/kubernetes-sigs/controller-runtime/commit/e59161ee8f41b2f24953419533dca84d30262c21#diff-d500fbd6a2aa620607ca5e2a7c3ac4f1a4c82309d1a549561e92abfcb18f2f0eR36.
		Metrics: metricsserver.Options{
			BindAddress:    metricsAddr,
			SecureServing:  true,
			TLSOpts:        tlsOpts,
			FilterProvider: filters.WithAuthenticationAndAuthorization,
		},
		// The old Port field was deprecated in:
		// https://github.com/kubernetes-sigs/controller-runtime/commit/f8c6ee427c500fb6ff8f8ee595110661c81da4e7#diff-d500fbd6a2aa620607ca5e2a7c3ac4f1a4c82309d1a549561e92abfcb18f2f0e.
		WebhookServer: &webhook.DefaultServer{
			Options: webhook.Options{
				Port: 9443,
			},
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0d697945.csi.netapp.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.BeegfsDriverReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("BeegfsDriver"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BeegfsDriver")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
