/*
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

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
	"k8s.io/apimachinery/pkg/types"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type OperatingSystemImage struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^https?://.*$"
	// +kubebuilder:validation:Format="url"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// Represents the operating system image URL.
	URL string `json:"url"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=64
	// +kubebuilder:validation:MaxLength=64
	// Represents the operating system image SHA256 sum.
	Sha256Sum string `json:"sha256sum"`

	// +kubebuilder:validation:Optional
	Force bool `json:"force"`
}

// HypervisorSpec defines the desired state of Hypervisor
type HypervisorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// OperatingSystemImage represents the desired operating system image.
	OperatingSystemImage *OperatingSystemImage `json:"image,omitempty"`
}

type Instance struct {
	// Represents the instance ID (uuidv4).
	ID string `json:"id"`

	// Represents the instance name.
	Name string `json:"name"`

	// Represents the instance state.
	Active bool `json:"active"`
}

// HypervisorStatus defines the observed state of Hypervisor
type HypervisorStatus struct {
	// Represents the Hypervisor version.
	Version string `json:"version"`

	// Represents the Hypervisor node name.
	Node types.NodeName `json:"node"`

	// Represents the Hypervisor hosted Virtual Machines
	Instances []Instance `json:"instances,omitempty"`

	// Represent the num of instances
	NumInstances int `json:"numInstances,omitempty"`

	// Represents the Hypervisor node conditions.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	SpecHash string `json:"specHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.version",name="Version",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.numInstances",name="Instances",type="integer"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Hypervisor is the Schema for the hypervisors API
type Hypervisor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HypervisorSpec   `json:"spec,omitempty"`
	Status HypervisorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HypervisorList contains a list of Hypervisor
type HypervisorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hypervisor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Hypervisor{}, &HypervisorList{})
}
