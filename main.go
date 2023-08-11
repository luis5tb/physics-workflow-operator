// Copyright 2021 Atos
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Created on 23 Mar 2022
// Updated on 23 Mar 2022
//
// @author: ATOS
package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"go.uber.org/zap/zapcore"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	zaplogfmt "github.com/sykesm/zap-logfmt"
	uzap "go.uber.org/zap"
	wp5v1alpha1 "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/api/v1alpha1"
	"gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/controllers" //+kubebuilder:scaffold:imports

	log "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/common/logs"
)

// path used in logs
const pathLOG string = "## [PHYSICS-WORKFLOW-OPERATOR] [Main] "

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	log.Debug(pathLOG + "[init] Initializing schemes ...")

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(wp5v1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	log.Info("[PHYSICS-WORKFLOW-OPERATOR; version: v0.0.1; date: 2023.08.11; id: 1]")

	log.Debug(pathLOG + "  Getting OPERATOR environment variables...")
	log.Debug(pathLOG + "    PHYSICS_OW_PROXY_ENDPOINT .... " + os.Getenv("PHYSICS_OW_PROXY_ENDPOINT"))
	log.Debug(pathLOG + "    PHYSICS_ACTION_PROXY_IMAGE ... " + os.Getenv("PHYSICS_ACTION_PROXY_IMAGE"))
	log.Debug(pathLOG + "    PHYSICS_OW_ENDPOINT .......... " + os.Getenv("PHYSICS_OW_ENDPOINT"))
	log.Debug(pathLOG + "    PHYSICS_OW_AUTH .............. " + os.Getenv("PHYSICS_OW_AUTH"))
	log.Debug(pathLOG + "    PHYSICS_OW_API_VERSION ....... " + os.Getenv("PHYSICS_OW_API_VERSION"))
	log.Debug(pathLOG + "    PHYSICS_OW_NAMESPACE ......... " + os.Getenv("PHYSICS_OW_NAMESPACE"))

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	//ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	// Custom log format
	configLog := uzap.NewProductionEncoderConfig()
	configLog.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339Nano))
	}
	logfmtEncoder := zaplogfmt.NewEncoder(configLog)
	logger := zap.New(zap.UseFlagOptions(&opts), zap.UseDevMode(true), zap.WriteTo(os.Stdout), zap.Encoder(logfmtEncoder))
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5b5692f5.physics-faas.eu",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.WorkflowReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Workflow")
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
