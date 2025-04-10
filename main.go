/*
Copyright 2021.

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

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/openshift-kni/cluster-group-upgrades-operator/controllers"
	ranv1alpha1 "github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	ibguv1alpha1 "github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/imagebasedgroupupgrades/v1alpha1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	viewv1beta1 "github.com/stolostron/cluster-lifecycle-api/view/v1beta1"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	mwv1 "open-cluster-management.io/api/work/v1"
	mwv1alpha1 "open-cluster-management.io/api/work/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(clusterv1.AddToScheme(scheme))
	utilruntime.Must(mwv1.AddToScheme(scheme))
	utilruntime.Must(mwv1alpha1.AddToScheme(scheme))
	utilruntime.Must(policiesv1.AddToScheme(scheme))
	utilruntime.Must(ranv1alpha1.AddToScheme(scheme))
	utilruntime.Must(ibguv1alpha1.AddToScheme(scheme))
	utilruntime.Must(viewv1beta1.AddToScheme(scheme))
	utilruntime.Must(actionv1beta1.AddToScheme(scheme))
	utilruntime.Must(operatorsv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableHTTP2 bool
	var metricsCertDir string

	flag.BoolVar(&enableHTTP2, "enable-http2", enableHTTP2, "If HTTP/2 should be enabled for the webhook server.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	flag.StringVar(&metricsCertDir, "metrics-tls-cert-dir", "",
		"The directory containing the tls.crt and tls.key.")
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Set the TLS options.
	// If the enable-http2 flag is false (the default), http/2 will be disabled due to its vulnerabilities.
	// More specifically, disabling http/2 will prevent from being vulnerable to the HTTP/2 Stream
	// Cancelation and Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			c.NextProtos = []string{"http/1.1"}
		})
	}

	ibguLabelReq, err := labels.NewRequirement(
		"openshift-cluster-group-upgrades/clusterGroupUpgrade", selection.Exists, []string{})
	if err != nil {
		setupLog.Error(err, "bad label selector")
		os.Exit(1)
	}
	selector := labels.NewSelector().Add(*ibguLabelReq)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9a2365a3.openshift.io",
		Metrics: server.Options{
			BindAddress:    metricsAddr,
			SecureServing:  metricsCertDir != "",
			CertDir:        metricsCertDir,
			TLSOpts:        tlsOpts,
			FilterProvider: filters.WithAuthenticationAndAuthorization,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    9443,
			TLSOpts: tlsOpts,
		}),
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&mwv1.ManifestWork{}: {
					Label: selector,
				},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterGroupUpgradeReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterGroupUpgrade"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterGroupUpgrade")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err = (&controllers.ManagedClusterForCguReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ManagedClusterForCGU"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ManagedClusterForCGU")
		os.Exit(1)
	}

	if err = (&controllers.IBGUReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ImageBasedGroupUpgrade"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageBasedGroupUpgrade")
		os.Exit(1)
	}

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
