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
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcore-dev/kvm-node-agent/internal/libvirt"
)

func NewLibVirtEmulator(ctx context.Context) *libvirt.InterfaceMock {
	log := logger.FromContext(ctx, "controller", "libvirt-emulator")
	mockedInterface := &libvirt.InterfaceMock{
		CloseFunc: func() error {
			log.Info("CloseFunc called")
			return nil
		},
		ConnectFunc: func() error {
			log.Info("Connect Func called")
			return nil
		},
		ProcessFunc: func(hv v1.Hypervisor) (v1.Hypervisor, error) {
			log.Info("Process Func called")
			return hv, nil
		},
	}
	return mockedInterface
}
