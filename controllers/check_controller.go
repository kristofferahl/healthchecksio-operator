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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/mitchellh/hashstructure"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	healthchecksio "github.com/kristofferahl/go-healthchecksio"
	monitoringv1alpha1 "github.com/kristofferahl/healthchecksio-operator/api/v1alpha1"
)

const (
	finalizerName = "check.finalizers.monitoring.healthchecks.io"
)

// CheckReconciler reconciles a Check object
type CheckReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	Hckio  *healthchecksio.Client
	Clock  Clock
}

// Clock enables mocking of time
type Clock struct {
	Source func() *metav1.Time
}

// Now returns the current time of the Source
func (c *Clock) Now() *metav1.Time {
	return c.Source()
}

// NewClock creates a new Clock
func NewClock() Clock {
	return Clock{
		Source: func() *metav1.Time {
			now := metav1.Now()
			return &now
		},
	}
}

// +kubebuilder:rbac:groups=monitoring.healthchecks.io,resources=checks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.healthchecks.io,resources=checks/status,verbs=get;update;patch

// Reconcile tries to reconcile the object
func (r *CheckReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("check", req.NamespacedName)

	var check monitoringv1alpha1.Check
	if err := r.Get(ctx, req.NamespacedName, &check); err != nil {
		log.V(0).Info("unable to fetch Check from k8s")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}
	log.V(1).Info("fetched Check from k8s")

	// examine DeletionTimestamp to determine if object is under deletion
	if check.ObjectMeta.DeletionTimestamp.IsZero() {
		log.V(1).Info("the Check is not being deleted, ensuring finalizer is present")
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(check.ObjectMeta.Finalizers, finalizerName) {
			check.ObjectMeta.Finalizers = append(check.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &check); err != nil {
				return ctrl.Result{}, err
			}
			log.V(1).Info("added finalizer for Check")
		}
	} else {
		log.V(1).Info("the Check is being deleted, checking if finalizer is present")
		// The object is being deleted
		if containsString(check.ObjectMeta.Finalizers, finalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(&check); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			log.V(0).Info(fmt.Sprintf("deleted healthcheck: %s", check.Status.ID))

			// remove our finalizer from the list and update it.
			check.ObjectMeta.Finalizers = removeString(check.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &check); err != nil {
				return ctrl.Result{}, err
			}
			log.V(0).Info("removed finalizer for Check")
		}

		return ctrl.Result{}, nil
	}

	channels := make([]string, 0)
	if len(check.Spec.Channels) > 0 {
		allChannels, err := r.Hckio.GetAllChannels()
		if err != nil {
			log.Error(err, "healthchecksio returned an error when fetching channels")
			return ctrl.Result{}, err
		}
		channels = matchTargetChannels(check, allChannels...)
		log.V(1).Info("fetched channels from healthchecksio")
	}

	healthcheck, err := r.Hckio.Create(convertToHealthcheck(check, channels...))
	if err != nil {
		log.Error(err, "healthchecksio returned an error when creating/updating healthcheck")
		return ctrl.Result{}, err
	}
	log.V(0).Info(fmt.Sprintf("created/updated healthcheck: %s", healthcheck.ID()))
	log.V(2).Info(fmt.Sprintf("healthcheck %s, %v", healthcheck.ID(), healthcheck))

	// Update the status based on the response
	if r.updateCheckStatus(&check, *healthcheck) {
		if err := r.Status().Update(ctx, &check); err != nil {
			log.Error(err, "unable to update Check status")
			return ctrl.Result{}, err
		}
		log.V(1).Info("updated the Check status")
	} else {
		log.V(1).Info("skipped update of the Check status")
	}

	// TODO: requeue configurable or not at all?
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func convertToHealthcheck(check monitoringv1alpha1.Check, channels ...string) healthchecksio.Healthcheck {
	timeout := 0
	if check.Spec.Timeout != nil {
		timeout = int(*check.Spec.Timeout)
	}

	graceperiod := 0
	if check.Spec.GracePeriod != nil {
		graceperiod = int(*check.Spec.GracePeriod)
	}

	return healthchecksio.Healthcheck{
		Name:     fmt.Sprintf("%s/%s", check.Namespace, check.Name),
		Schedule: check.Spec.Schedule,
		Timezone: check.Spec.Timezone,
		Timeout:  timeout,
		Grace:    graceperiod,
		Tags:     strings.Join(check.Spec.Tags, " "),
		Channels: strings.Join(channels, ","),
		Unique:   []string{"name"},
	}
}

func matchTargetChannels(check monitoringv1alpha1.Check, allChannels ...*healthchecksio.HealthcheckChannelResponse) []string {
	channels := make([]string, 0)

	if len(check.Spec.Channels) == 1 && check.Spec.Channels[0] == "*" {
		channels = append(channels, "*")
	} else if len(check.Spec.Channels) > 0 {
		for _, channelKindName := range check.Spec.Channels {
			p := strings.Split(channelKindName, "/")
			kind := p[0]
			name := ""
			if len(p) == 2 {
				name = p[1]
			}

			for _, c := range allChannels {
				if isTargetChannel(c, name, kind) {
					channels = append(channels, c.ID)
				}
			}
		}
	}

	return channels
}

func (r *CheckReconciler) updateCheckStatus(check *monitoringv1alpha1.Check, healthcheck healthchecksio.HealthcheckResponse) bool {
	changed := false

	before, err := hashstructure.Hash(check.Status, nil)
	if err != nil {
		changed = true
	}

	pings := int32(healthcheck.Pings)

	check.Status.ObservedGeneration = check.ObjectMeta.Generation
	check.Status.ID = healthcheck.ID()
	check.Status.PingURL = healthcheck.PingURL
	check.Status.Status = healthcheck.Status
	check.Status.Pings = &pings
	check.Status.LastPing = parseTimestamp(healthcheck.LastPing)

	after, err := hashstructure.Hash(check.Status, nil)
	if err != nil {
		changed = true
	}

	if changed == false {
		changed = before != after
	}

	if changed {
		check.Status.LastUpdated = r.Clock.Now()
	}

	return changed
}

// Delete any external resources associated with the check.
// Ensure that delete implementation is idempotent and safe to
// invoke multiple times for same object.
func (r *CheckReconciler) deleteExternalResources(check *monitoringv1alpha1.Check) error {
	_, err := r.Hckio.Delete(check.Status.ID)
	if err != nil {
		if err, ok := err.(*healthchecksio.APIError); ok && err.StatusCode() == 404 {
			r.Log.V(1).Info(fmt.Sprintf("healthcheck not found or already deleted (status=%s)", err.Status()))
			return nil
		}
		return err
	}
	return nil
}

func parseTimestamp(ts string) *metav1.Time {
	var lastPing metav1.Time
	lp, err := time.Parse(time.RFC3339, ts)
	if err == nil {
		lastPing = metav1.NewTime(lp)
		return &lastPing
	}

	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func isTargetChannel(c *healthchecksio.HealthcheckChannelResponse, name, kind string) bool {
	if name == "" {
		return c.Kind == kind
	}

	return c.Name == name && c.Kind == kind
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

// SetupWithManager hooks up the controller/reconciler
func (r *CheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Check{}).
		Complete(r)
}
