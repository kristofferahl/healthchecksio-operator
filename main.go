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
	scheme      = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	apiKey      = os.Getenv("HEALTHCHECKSIO_API_KEY")
	development = os.Getenv("HEALTHCHECKSIO_OPERATOR_DEVELOPMENT")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = monitoringv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var reconcileInterval time.Duration
	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.DurationVar(&reconcileInterval, "reconcile-interval", 1*time.Minute, "The interval for the reconcile loop")
	flag.StringVar(&logLevel, "log-level", "info", "The log level to use.")
	flag.Parse()

	ctrl.SetLogger(logrzap.New(func(o *logrzap.Options) {
		dev, err := strconv.ParseBool(development)
		if err == nil {
			o.Development = dev
		}

		if o.Development == false {
			lev := zap.NewAtomicLevel()
			(&lev).UnmarshalText([]byte(logLevel))
			o.Level = &lev
		}
	}))

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
