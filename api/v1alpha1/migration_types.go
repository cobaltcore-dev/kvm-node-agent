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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MigrationSpec defines the desired state of Migration.
type MigrationSpec struct {
}

// MigrationStatus defines the observed state of Migration.
type MigrationStatus struct {
	Type                 string `json:"type"`
	ErrMsg               string `json:"errMsg,omitempty"`
	AutoConvergeThrottle string `json:"autoConvergeThrottle,omitempty"`
	DiskBps              string `json:"diskBps,omitempty"`
	DiskRemaining        string `json:"diskRemaining,omitempty"`
	DiskProcessed        string `json:"diskProcessed,omitempty"`
	DiskTotal            string `json:"diskTotal,omitempty"`
	MemPostcopyRequests  uint64 `json:"memPostcopyRequests,omitempty"`
	MemIteration         uint64 `json:"memIteration,omitempty"`
	MemPageSize          string `json:"memPageSize,omitempty"`
	MemDirtyRate         string `json:"memDirtyRate,omitempty"`
	MemBps               string `json:"memBps,omitempty"`
	MemNormalBytes       string `json:"memNormalBytes,omitempty"`
	MemNormal            uint64 `json:"memNormal,omitempty"`
	MemConstant          uint64 `json:"memConstant,omitempty"`
	MemRemaining         string `json:"memRemaining,omitempty"`
	MemProcessed         string `json:"memProcessed,omitempty"`
	MemTotal             string `json:"memTotal,omitempty"`
	DataRemaining        string `json:"dataRemaining,omitempty"`
	DataProcessed        string `json:"dataProcessed,omitempty"`
	DataTotal            string `json:"dataTotal,omitempty"`
	SetupTime            string `json:"setupTime,omitempty"`
	TimeElapsed          string `json:"timeElapsed,omitempty"`
	TimeRemaining        string `json:"timeRemaining,omitempty"`
	Downtime             string `json:"downtime,omitempty"`
	Operation            string `json:"operation,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.status.type`
// +kubebuilder:printcolumn:name="Operation",type=string,JSONPath=`.status.operation`
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Elapsed",type=string,JSONPath=`.status.timeElapsed`
// +kubebuilder:printcolumn:name="Remaining",type=string,JSONPath=`.status.timeRemaining`
// +kubebuilder:printcolumn:name="Data Total",type=string,JSONPath=`.status.dataTotal`
// +kubebuilder:printcolumn:name="Data Processed",type=string,JSONPath=`.status.dataProcessed`
// +kubebuilder:printcolumn:name="Data Remaining",type=string,JSONPath=`.status.dataRemaining`
// +kubebuilder:printcolumn:name="Memory Bps",type=string,JSONPath=`.status.memBps`
// +kubebuilder:printcolumn:name="Memory Dirty Rate",type=string,JSONPath=`.status.memDirtyRate`
// +kubebuilder:printcolumn:name="Memory Iteration",type=string,JSONPath=`.status.memIteration`

// Migration is the Schema for the migrations API.
type Migration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MigrationSpec   `json:"spec,omitempty"`
	Status MigrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MigrationList contains a list of Migration.
type MigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Migration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Migration{}, &MigrationList{})
}
