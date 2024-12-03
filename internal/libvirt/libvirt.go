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

package libvirt

import (
	"fmt"
	"os"

	"github.com/cobaltcode-dev/kvm-node-agent/api/v1alpha1"
	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LibVirt struct {
	virt *libvirt.Libvirt
}

func NewLibVirt() *LibVirt {
	socketPath := os.Getenv("LIBVIRT_SOCKET")
	if socketPath == "" {
		socketPath = "/run/libvirt/libvirt-sock"
	}
	log.Log.Info("Using libvirt unix domain socket", "socket", socketPath)
	l := &LibVirt{libvirt.NewWithDialer(dialers.NewLocal(dialers.WithSocket(socketPath)))}
	defer func() {
		l.statsCollector()
	}()
	return l
}

func (l *LibVirt) Connect() error {
	return l.virt.Connect()
}

func (l *LibVirt) GetVersion() (string, error) {
	var version uint64
	var err error
	if version, err = l.virt.ConnectGetVersion(); err != nil {
		return "", fmt.Errorf("failed to fetch libvirt version: %w", err)
	}
	major, minor, release := version/1000000, (version/1000)%1000, version%1000
	return fmt.Sprintf("%d.%d.%d", major, minor, release), nil
}

func (l *LibVirt) Close() error {
	return l.virt.Disconnect()
}

func (l *LibVirt) GetInstances() ([]v1alpha1.Instance, error) {
	var instances []v1alpha1.Instance
	//expose everything what is possible on connectlistdomnainsactive for prometheus
	flags := []libvirt.ConnectListAllDomainsFlags{libvirt.ConnectListDomainsActive, libvirt.ConnectListDomainsInactive}
	for _, flag := range flags {
		domains, _, err := l.virt.ConnectListAllDomains(1, flag)
		if err != nil {
			return nil, err
		}

		for _, domain := range domains {
			var u uuid.UUID
			u, err = uuid.FromBytes(domain.UUID[:])
			if err != nil {
				return nil, err
			}

			instances = append(instances, v1alpha1.Instance{
				ID:     u.String(),
				Name:   domain.Name,
				Active: flag == libvirt.ConnectListDomainsActive,
			})
		}
	}
	return instances, nil
}

func (l *LibVirt) IsConnected() bool {
	return l.virt.IsConnected()
}
