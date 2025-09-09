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

//go:generate moq -out libvirt_mock.go . Interface

package libvirt

import (
	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	"github.com/digitalocean/go-libvirt"
)

type Interface interface {
	// Connect connects to the libvirt daemon.
	Connect() error

	// Close closes the connection to the libvirt daemon.
	Close() error

	// GetInstances returns a list of instances.
	GetInstances() ([]v1.Instance, error)

	// GetDomainsActive returns all active domains.
	GetDomainsActive() ([]libvirt.Domain, error)

	// IsConnected returns true if the connection to the libvirt daemon is open.
	IsConnected() bool

	// GetVersion returns the version of the libvirt daemon.
	GetVersion() string

	// GetNumInstances returns the number of instances.
	GetNumInstances() int

	// Get the capabilities of the libvirt daemon.
	GetCapabilities() (v1.CapabilitiesStatus, error)
}
