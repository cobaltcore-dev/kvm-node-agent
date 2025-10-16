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

package emulator

import (
	"context"

	v1 "github.com/cobaltcore-dev/openstack-hypervisor-operator/api/v1"
	"github.com/coreos/go-systemd/v22/dbus"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/systemd"
)

func NewSystemdEmulator(ctx context.Context) *systemd.InterfaceMock {
	log := logger.FromContext(ctx, "controller", "systemd-emulator")
	mockedInterface := &systemd.InterfaceMock{
		CloseFunc: func() {
			log.Info("CloseFunc called")
		},
		GetUnitByNameFunc: func(ctx context.Context, unit string) (dbus.UnitStatus, error) {
			log.Info("GetUnitByNameFunc called with unit = " + unit)
			return dbus.UnitStatus{}, nil
		},
		IsConnectedFunc: func() bool {
			log.Info("GetUnitByNameFunc called")
			return true
		},
		ListUnitsByNamesFunc: func(ctx context.Context, units []string) ([]dbus.UnitStatus, error) {
			log.Info("GetUnitByNameFunc called")
			return nil, nil
		},
		ReconcileSysUpdateFunc: func(ctx context.Context, hv *v1.Hypervisor) (bool, error) {
			log.Info("GetUnitByNameFunc called")
			return true, nil
		},
		StartUnitFunc: func(ctx context.Context, unit string) (int, error) {
			log.Info("GetUnitByNameFunc called")
			return 0, nil
		},
		EnableShutdownInhibitFunc: func(ctx context.Context, cb func(ctx context.Context) error) error {
			log.Info("GetUnitByNameFunc called")
			return nil
		},
		DisableShutdownInhibitFunc: func() error {
			log.Info("GetUnitByNameFunc called")
			return nil
		},
	}
	return mockedInterface
}
