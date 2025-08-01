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

// AuthMethodSpec defines the desired state of AuthMethod.
type AuthMethodSpec struct {
	AuthMethods []AuthMethodEntry `json:"authMethods"`
}

type AuthMethodEntry struct {
	Type  string           `json:"type"`
	Path  string           `json:"path"`
	Roles []AuthMethodRole `json:"roles,omitempty"`
}

type AuthMethodRole struct {
	Name           string   `json:"name"`
	ServiceAccount string   `json:"serviceaccount"`
	Namespaces     []string `json:"namespaces"`
	Policies       []string `json:"policies"`
}

// AuthMethodStatus defines the observed state of AuthMethod.
type AuthMethodStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AuthMethod is the Schema for the authmethods API.
type AuthMethod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthMethodSpec   `json:"spec,omitempty"`
	Status AuthMethodStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AuthMethodList contains a list of AuthMethod.
type AuthMethodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthMethod `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthMethod{}, &AuthMethodList{})
}
