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

// RunnerImageSpec defines the desired state of RunnerImage
type RunnerImageSpec struct {
	Plugins Plugins `json:"plugins,omitempty"`
}

type Plugins []string

// RunnerImageStatus defines the observed state of RunnerImage
type RunnerImageStatus struct {
	// Phase would be either of Initialized, Started, Completed, Error
	// +optional
	Phase string `json:"phase,omitempty"`
	// Message is the message obtained at a certain state
	// +optional
	Message string `json:"message,omitempty"`
	// Reason would be used when there is an error and would be either of
	// +optional
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Plugins",type=string,JSONPath=`.spec.plugins`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// RunnerImage is the Schema for the runnerimages API
type RunnerImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerImageSpec   `json:"spec,omitempty"`
	Status RunnerImageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunnerImageList contains a list of RunnerImage
type RunnerImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunnerImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunnerImage{}, &RunnerImageList{})
}
