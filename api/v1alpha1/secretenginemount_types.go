/*
Copyright 2025.

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

// SecretEngineMountSpec defines the desired state of SecretEngineMount.
type SecretEngineMountSpec struct {
	Mounts []Mount `json:"mounts"`
}

type Mount struct {
	Type        string            `json:"type"`
	Path        string            `json:"path"`
	Description string            `json:"description,omitempty"`
	Options     map[string]string `json:"options,omitempty"`
}

// SecretEngineMountStatus defines the observed state of SecretEngineMount.
type SecretEngineMountStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Mounted []MountStatus `json:"mounted,omitempty"`
}

type MountStatus struct {
	Path    string `json:"path"`
	Mounted bool   `json:"mounted"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SecretEngineMount is the Schema for the secretenginemounts API.
type SecretEngineMount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretEngineMountSpec   `json:"spec,omitempty"`
	Status SecretEngineMountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretEngineMountList contains a list of SecretEngineMount.
type SecretEngineMountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretEngineMount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretEngineMount{}, &SecretEngineMountList{})
}
