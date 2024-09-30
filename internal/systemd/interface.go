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

//go:generate moq -out systemd_mock.go . Interface

package systemd

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
)

type Interface interface {
	// Close closes the connection to the systemd D-Bus API.
	Close()

	// IsConnected returns true if the connection to the systemd D-Bus API is open.
	IsConnected() bool

	// ListUnitsByNames returns the status of the units with the given names.
	ListUnitsByNames(ctx context.Context, units []string) ([]dbus.UnitStatus, error)

	// GetUnitByName returns the status of the unit with the given name.
	GetUnitByName(ctx context.Context, unit string) (dbus.UnitStatus, error)

	// StartUnit starts the unit with the given name.
	StartUnit(ctx context.Context, unit string) (int, error)

	// ReconcileSysUpdate reconciles orchestrates a systemd-sysupdate via the systemd-sysupdate@.service unit.
	ReconcileSysUpdate(ctx context.Context, hv *v1alpha1.Hypervisor) (bool, error)
}
