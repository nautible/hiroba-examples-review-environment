/*
Copyright 2023.

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

// MergeRequestSpec defines the desired state of MergeRequest
type MergeRequestSpec struct {
	Name           string `json:"name"`                     // GitLab group
	Application    string `json:"application"`              // GitLab project
	BaseUrl        string `json:"baseUrl"`                  // GitLab Base URL
	ManifestPath   string `json:"manifestPath,omitempty"`   // manifests root path
	TargetRevision string `json:"targetRevision,omitempty"` // Application TargetRevision
}

// MergeRequestStatus defines the observed state of MergeRequest
type MergeRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MergeRequest is the Schema for the mergerequests API
type MergeRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MergeRequestSpec   `json:"spec,omitempty"`
	Status MergeRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MergeRequestList contains a list of MergeRequest
type MergeRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MergeRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MergeRequest{}, &MergeRequestList{})
}
