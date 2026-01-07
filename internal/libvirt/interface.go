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
)

type Interface interface {
	// Connect connects to the libvirt daemon.
	Connect() error

	// Close closes the connection to the libvirt daemon.
	Close() error

	// Add information extracted from the libvirt socket to the hypervisor instance.
	// If an error occurs, the instance is returned unmodified. The libvirt
	// connection needs to be established before calling this function.
	Process(hv v1.Hypervisor) (v1.Hypervisor, error)
}
