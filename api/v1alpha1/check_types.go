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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CheckSpec defines the desired state of Check
type CheckSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=0

	// The schedule in Cron format
	Schedule string `json:"schedule,omitempty"`

	// +kubebuilder:validation:MinLength=0

	// Server's timezone. This setting only has effect in combination with the "schedule" property.
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// A number of seconds, the expected period of the check.
	// +optional
	Timeout *int32 `json:"timeout,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// A number of seconds, the grace period for the check.
	// +optional
	GracePeriod *int32 `json:"gracePeriod,omitempty"`

	// +kubebuilder:validation:MinItems=0

	// A list of tags for the check.
	// +optional
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:MinItems=0

	// A list of channels to assign to the check.
	// +optional
	Channels []string `json:"channels,omitempty"`
}

// CheckStatus defines the observed state of Check
type CheckStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The ID of the check
	// +optional
	ID string `json:"id,omitempty"`

	// When was the last time the check was successfully updated.
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// What was the status of the check.
	// +optional
	Status string `json:"status,omitempty"`

	// What number of times has the check been pinged.
	// +optional
	Pings *int32 `json:"pings,omitempty"`

	// When was the last time the check was successfully pinged.
	// +optional
	LastPing *metav1.Time `json:"lastPing,omitempty"`

	// The URL used for pinging the check
	// +optional
	PingURL string `json:"pingURL,omitempty"`

	// The last seen generation of the resource
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Timezone",priority=0,type=string,JSONPath=`.spec.timezone`
// +kubebuilder:printcolumn:name="GracePeriod",type=integer,JSONPath=`.spec.gracePeriod`
// +kubebuilder:printcolumn:name="Status",priority=0,type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Pings",priority=1,type=integer,JSONPath=`.status.pings`
// +kubebuilder:printcolumn:name="LastPing",priority=1,type=string,format="date-time",JSONPath=`.status.lastPing`
// +kubebuilder:printcolumn:name="LastUpdated",priority=1,type=string,format="date-time",JSONPath=`.status.lastUpdated`

// Check is the Schema for the checks API
type Check struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CheckSpec   `json:"spec,omitempty"`
	Status CheckStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CheckList contains a list of Check
type CheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Check `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Check{}, &CheckList{})
}
