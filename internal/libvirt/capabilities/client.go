/*
SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
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

package capabilities

import (
	"encoding/xml"
	"fmt"

	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	libvirt "github.com/digitalocean/go-libvirt"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client that returns the capabilities of the host we are mounted on.
type Client interface {
	// Return the capabilities status of the host we are mounted on.
	Get(virt *libvirt.Libvirt) (v1.CapabilitiesStatus, error)
}

// Implementation of the CapabilitiesClient interface.
type client struct{}

// Create a new capabilities client.
func NewClient() Client {
	return &client{}
}

// Return the capabilities of the host we are mounted on.
func (m *client) Get(virt *libvirt.Libvirt) (v1.CapabilitiesStatus, error) {
	capabilitiesXMLBytes, err := virt.Capabilities()
	if err != nil {
		log.Log.Error(err, "failed to get libvirt capabilities")
		return v1.CapabilitiesStatus{}, err
	}
	var capabilities Capabilities
	if err := xml.Unmarshal(capabilitiesXMLBytes, &capabilities); err != nil {
		log.Log.Error(err, "failed to unmarshal libvirt capabilities")
		return v1.CapabilitiesStatus{}, err
	}
	return convert(capabilities)
}

// Emulated capabilities client returning an embedded capabilities xml.
type clientEmulator struct{}

// Create a new emulated capabilities client.
func NewClientEmulator() Client {
	return &clientEmulator{}
}

// Get the capabilities of the host we are mounted on.
func (c *clientEmulator) Get(virt *libvirt.Libvirt) (v1.CapabilitiesStatus, error) {
	var capabilities Capabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		log.Log.Error(err, "failed to unmarshal example capabilities")
		return v1.CapabilitiesStatus{}, err
	}
	return convert(capabilities)
}

// Convert the libvirt capabilities to the API format.
func convert(in Capabilities) (out v1.CapabilitiesStatus, err error) {
	out.HostCpuArch = in.Host.CPU.Arch
	// Loop over all numa cells to get the total memory + vcpus.
	totalMemory := resource.NewQuantity(0, resource.BinarySI)
	totalCpus := resource.NewQuantity(0, resource.DecimalSI)
	for _, cell := range in.Host.Topology.CellSpec.Cells {
		mem, err := cell.Memory.AsQuantity()
		if err != nil {
			return v1.CapabilitiesStatus{}, err
		}
		totalMemory.Add(mem)
		cpu := resource.NewQuantity(cell.CPUs.Num, resource.DecimalSI)
		if cpu == nil {
			return v1.CapabilitiesStatus{},
				fmt.Errorf("invalid CPU count for cell %d", cell.ID)
		}
		totalCpus.Add(*cpu)
	}
	out.HostMemory = *totalMemory
	out.HostCpus = *totalCpus
	return out, nil
}
