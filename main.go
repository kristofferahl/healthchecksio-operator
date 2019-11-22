/*

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
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	logr "github.com/go-logr/logr"
	healthchecksio "github.com/kristofferahl/go-healthchecksio"
	monitoringv1alpha1 "github.com/kristofferahl/healthchecksio-operator/api/v1alpha1"
	"github.com/kristofferahl/healthchecksio-operator/controllers"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	logrzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = monitoringv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var apiKey string
	var metricsAddr string
	var enableLeaderElection bool
	var development bool
	var logLevel string
	var namePrefix string
	var reconcileInterval time.Duration

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&development, "development", false, "Run the operator in development mode.")
	flag.StringVar(&logLevel, "log-level", "info", "The log level used by the operator.")
	flag.StringVar(&namePrefix, "name-prefix", "", "Prefix used to create unique resources across clusters.")
	flag.DurationVar(&reconcileInterval, "reconcile-interval", 1*time.Minute, "The interval for the reconcile loop")
	flag.Parse()

	apiKey = envOrDefaultString("HEALTHCHECKSIO_API_KEY", "")
	metricsAddr = envOrDefaultString("OPERATOR_METRICS_ADDR", metricsAddr)
	enableLeaderElection = envOrDefaultBool("OPERATOR_ENABLE_LEADER_ELECTION", enableLeaderElection)
	development = envOrDefaultBool("OPERATOR_DEVELOPMENT", development)
	logLevel = envOrDefaultString("OPERATOR_LOG_LEVEL", logLevel)
	namePrefix = envOrDefaultString("OPERATOR_NAME_PREFIX", namePrefix)
	reconcileInterval = envOrDefaultDuration("OPERATOR_RECONCILE_INTERVAL", reconcileInterval)

	ctrl.SetLogger(logrzap.New(func(o *logrzap.Options) {
		o.Development = development

		if o.Development == false {
			lev := zap.NewAtomicLevel()
			(&lev).UnmarshalText([]byte(logLevel))
			o.Level = &lev
		}
	}))

	setupLog.Info(
		"configuration",
		"metricsAddr", metricsAddr,
		"enableLeaderElection", enableLeaderElection,
		"development", development,
		"logLevel", logLevel,
		"namePrefix", namePrefix,
		"reconcileInterval", reconcileInterval,
	)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	hckioClient := healthchecksio.NewClient(apiKey)
	hckioClient.Log = &logrLogger{
		log: ctrl.Log.WithName("hckio-client"),
	}

	if err = (&controllers.CheckReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Log:               ctrl.Log.WithName("controllers").WithName("Check"),
		Hckio:             hckioClient,
		Clock:             controllers.NewClock(),
		ReconcileInterval: reconcileInterval,
		NamePrefix:        namePrefix,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Check")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func envOrDefaultString(key, defaultValue string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return defaultValue
}

func envOrDefaultBool(key string, defaultValue bool) bool {
	v := envOrDefaultString(key, strconv.FormatBool(defaultValue))
	pv, err := strconv.ParseBool(v)
	if err != nil {
		log.Panicf("failed parsing boolean from environment variable %s", key)
	}
	return pv
}

func envOrDefaultDuration(key string, defaultValue time.Duration) time.Duration {
	v := envOrDefaultString(key, defaultValue.String())
	pv, err := time.ParseDuration(v)
	if err != nil {
		log.Panicf("failed parsing duration from environment variable %s", key)
	}
	return pv
}

type logrLogger struct {
	log logr.Logger
}

func (l *logrLogger) Debugf(format string, args ...interface{}) {
	l.log.V(1).Info(fmt.Sprintf(format, args...))
}

func (l *logrLogger) Infof(format string, args ...interface{}) {
	l.log.V(0).Info(fmt.Sprintf(format, args...))
}

func (l *logrLogger) Errorf(format string, args ...interface{}) {
	l.log.Error(nil, fmt.Sprintf(format, args...))
}
