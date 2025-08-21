/*
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, LibVirtVersion 2.0 (the "License");
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
// Important: Run "make" to regenerate code after modifying this file

// HypervisorSpec defines the desired state of Hypervisor
type HypervisorSpec struct {
	// +kubebuilder:validation:Optional
	// OperatingSystemVersion represents the desired operating system version.
	OperatingSystemVersion string `json:"version,omitempty"`

	// +kubebuilder:default:=false
	// Reboot request an reboot after successful installation of an upgrade.
	Reboot bool `json:"reboot"`

	// +kubebuilder:default:=true
	// EvacuateOnReboot request an evacuation of all instances before reboot.
	EvacuateOnReboot bool `json:"evacuateOnReboot"`

	// +kubebuilder:default:=false
	// Require to issue a certificate from cert-manager for the hypervisor, to be used for
	// secure communication with the libvirt API.
	CreateCertManagerCertificate bool `json:"createCertManagerCertificate"`
}

type Instance struct {
	// Represents the instance ID (uuidv4).
	ID string `json:"id"`

	// Represents the instance name.
	Name string `json:"name"`

	// Represents the instance state.
	Active bool `json:"active"`
}

type HyperVisorUpdateStatus struct {
	// +kubebuilder:default:=false
	// Represents a running Operating System update.
	InProgress bool `json:"inProgress"`

	// +kubebuilder:default:=unknown
	// Represents the Operating System installed update version.
	Installed string `json:"installed,omitempty"`

	// +kubebuilder:default:=3
	// Represents the number of retries.
	Retry int `json:"retry"`
}

type OperatingSystemStatus struct {
	// Represents the Operating System version.
	Version string `json:"version,omitempty"`

	// PrettyVersion
	PrettyVersion string `json:"prettyVersion,omitempty"`

	// KernelName
	KernelName string `json:"kernelName,omitempty"`

	// KernelRelease
	KernelRelease string `json:"kernelRelease,omitempty"`

	// KernelVersion
	KernelVersion string `json:"kernelVersion,omitempty"`

	// HardwareVendor
	HardwareVendor string `json:"hardwareVendor,omitempty"`

	// HardwareModel
	HardwareModel string `json:"hardwareModel,omitempty"`

	// HardwareSerial
	HardwareSerial string `json:"hardwareSerial,omitempty"`

	// FirmwareVersion
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// FirmwareVendor
	FirmwareVendor string `json:"firmwareVendor,omitempty"`

	// FirmwareDate
	FirmwareDate metav1.Time `json:"firmwareDate,omitempty"`
}

// Current capabilities reported by libvirt.
type CapabilitiesStatus struct {
	// +kubebuilder:default:=unknown
	// The hosts CPU architecture (not the guests).
	HostCpuArch string `json:"cpuArch,omitempty"`
	// Total host memory available as a sum of memory over all numa cells.
	HostMemory resource.Quantity `json:"memory,omitempty"`
	// Total host cpus available as a sum of cpus over all numa cells.
	HostCpus resource.Quantity `json:"cpus,omitempty"`
}

// HypervisorStatus defines the observed state of Hypervisor
type HypervisorStatus struct {
	// +kubebuilder:default:=unknown
	// Represents the LibVirt version.
	LibVirtVersion string `json:"libVirtVersion"`

	// Represents the Operating System status.
	OperatingSystem OperatingSystemStatus `json:"operatingSystem,omitempty"`

	// Represents the Hypervisor update status.
	Update HyperVisorUpdateStatus `json:"updateStatus"`

	// Represents the Hypervisor node name.
	Node types.NodeName `json:"node"`

	// Represents the Hypervisor hosted Virtual Machines
	Instances []Instance `json:"instances,omitempty"`

	// The capabilities of the hypervisors as reported by libvirt.
	Capabilities CapabilitiesStatus `json:"capabilities,omitempty"`

	// +kubebuilder:default:=0
	// Represent the num of instances
	NumInstances int `json:"numInstances"`

	// Represents the Hypervisor node conditions.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	SpecHash string `json:"specHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.node",name="Node",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.operatingSystem.prettyVersion",name="Version",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.numInstances",name="Instances",type="integer"
// +kubebuilder:printcolumn:JSONPath=".status.operatingSystem.hardwareModel",name="Hardware",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.operatingSystem.kernelRelease",name="Kernel",type="string"
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
