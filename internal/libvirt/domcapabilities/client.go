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

package domcapabilities

import (
	"encoding/xml"

	libvirt "github.com/digitalocean/go-libvirt"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client that returns the domain capabilities of the host we are mounted on.
type Client interface {
	// Return the capabilities status of the host we are mounted on.
	Get(virt *libvirt.Libvirt) (DomainCapabilities, error)
}

// Implementation of the Client interface.
type client struct{}

// Create a new domain capabilities client.
func NewClient() Client {
	return &client{}
}

// Return the domain capabilities of the host we are mounted on.
func (m *client) Get(virt *libvirt.Libvirt) (DomainCapabilities, error) {
	// Same as running `virsh domcapabilities` without any arguments.
	capabilitiesXMLStr, err := virt.
		ConnectGetDomainCapabilities(nil, nil, nil, nil, 0)
	if err != nil {
		log.Log.Error(err, "failed to get libvirt capabilities")
		return DomainCapabilities{}, err
	}
	var capabilities DomainCapabilities
	if err := xml.Unmarshal([]byte(capabilitiesXMLStr), &capabilities); err != nil {
		log.Log.Error(err, "failed to unmarshal libvirt capabilities")
		return DomainCapabilities{}, err
	}
	return capabilities, nil
}

// Emulated domain capabilities client returning an embedded capabilities xml.
type clientEmulator struct{}

// Create a new emulated domain capabilities client.
func NewClientEmulator() Client {
	return &clientEmulator{}
}

// Get the domain capabilities of the host we are mounted on.
func (c *clientEmulator) Get(virt *libvirt.Libvirt) (DomainCapabilities, error) {
	var capabilities DomainCapabilities
	if err := xml.Unmarshal(exampleXML, &capabilities); err != nil {
		log.Log.Error(err, "failed to unmarshal example capabilities")
		return DomainCapabilities{}, err
	}
	return capabilities, nil
}
